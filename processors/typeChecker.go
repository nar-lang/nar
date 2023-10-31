package processors

import (
	"fmt"
	"maps"
	"oak-compiler/ast"
	"oak-compiler/ast/normalized"
	"oak-compiler/ast/typed"
	"oak-compiler/common"
	"os"
	"path/filepath"
	"strings"
)

var unboundIndex = uint64(0)
var typeMap map[fmt.Stringer]typed.Type

type symMap map[ast.Identifier]typed.Type
type typeParamsMap map[ast.Identifier]uint64

type equation struct {
	left, right typed.Type
	expr        typed.Expression
	pattern     typed.Pattern
	def         *typed.Definition
}

func (e equation) String() string {
	x := fmt.Stringer(e.expr)
	if x == nil {
		x = e.pattern
	}
	if x == nil {
		x = e.def
	}
	return fmt.Sprintf("\n| `%v` | `%v` | `%v` |", e.left, e.right, x)
}

func CheckTypes(
	path string, modules map[string]normalized.Module, typedModules map[string]*typed.Module,
) {
	if _, ok := typedModules[path]; ok {
		return
	}

	m := modules[path]

	for _, dep := range m.DepPaths {
		CheckTypes(dep, modules, typedModules)
	}

	o := typed.Module{
		Path:        m.Path,
		Definitions: map[ast.Identifier]*typed.Definition{},
	}

	typedModules[o.Path] = &o

	for name, def := range m.Definitions {
		if _, ok := o.Definitions[name]; !ok {
			unboundIndex = 0
			typeMap = map[fmt.Stringer]typed.Type{}
			localTyped := map[string]*typed.Module{}
			symtab := symMap{}
			var eqs []equation

			td := annotateDefinition(symtab, modules, localTyped, def)
			eqs = equatizeDefinition(eqs, td)
			o.Definitions[name] = td

			sb := strings.Builder{}
			sb.WriteString(fmt.Sprintf("\n\nDefinition\n---\n`%s`", td))
			sb.WriteString("\n\nAssignments\n---\n| Node | Type |\n|---|---|")
			for n, t := range typeMap {
				sb.WriteString(fmt.Sprintf("\n| `%v` | `%v` |", n, t))
			}
			sb.WriteString("\n\nEquations\n---\n| Left | Right | Node |\n|---|---|---|")
			for _, eq := range eqs {
				sb.WriteString(eq.String())
			}

			subst := unifyAll(eqs)

			sb.WriteString("\n\nUnified\n---\n| Left | Right |\n|---|---|")
			for k, v := range subst {
				sb.WriteString(fmt.Sprintf("\n | `%v` | `%v` |", &typed.TUnbound{Index: k}, v))
			}

			td = applyDefinition(td, subst)

			sb.WriteString("\n\nSolved\n---\n")
			sb.WriteString(fmt.Sprintf("\n `%v`", td.Type))

			cwd, _ := os.Getwd()
			rp, _ := filepath.Rel(cwd, m.Path[:len(m.Path)-4])
			p := strings.Replace(rp, "../", "", -1)

			fp := fmt.Sprintf(".oak-bin/%s/%s.md", p, td.Pattern.(*typed.PNamed).Name)
			_ = os.MkdirAll(filepath.Dir(fp), 0700)
			_ = os.WriteFile(fp, []byte(sb.String()), 0666)

		}
	}
}

func equatizeDefinition(eqs []equation, td *typed.Definition) []equation {
	if td.Expression != nil {
		eqs = append(eqs, equation{
			left:  td.Type,
			right: td.Expression.GetType(),
			def:   td,
		})
		eqs = equatizeExpression(eqs, td.Expression)
	}
	eqs = equatizePattern(eqs, td.Pattern)
	return eqs
}

func equatizePattern(eqs []equation, pattern typed.Pattern) []equation {
	switch pattern.(type) {
	case *typed.PAlias:
		{
			e := pattern.(*typed.PAlias)
			eqs = equatizePattern(eqs, e.Nested)
			break
		}
	case *typed.PAny:
		{
			break
		}
	case *typed.PCons:
		{
			e := pattern.(*typed.PCons)
			eqs = append(eqs,
				equation{
					left:    e.Type,
					right:   e.Tail.GetType(),
					pattern: pattern,
				},
				equation{
					left: e.Type,
					right: &typed.TExternal{
						Location: e.Location,
						Name:     common.OakCoreListList,
						Args:     []typed.Type{e.Head.GetType()},
					},
					pattern: pattern,
				})
			eqs = equatizePattern(eqs, e.Head)
			eqs = equatizePattern(eqs, e.Tail)
			break
		}
	case *typed.PConst:
		{
			e := pattern.(*typed.PConst)
			eqs = append(eqs, equation{
				left:    e.Type,
				right:   getConstType(e.Value, e.Location),
				pattern: pattern,
			})
			break
		}
	case *typed.PDataValue:
		{
			e := pattern.(*typed.PDataValue)
			tf := e.Definition.Type.(*typed.TFunc)
			eqs = append(eqs, equation{
				left:    e.Type,
				right:   tf.Return,
				pattern: pattern,
			})
			if len(e.Values) != len(tf.Params) {
				panic(common.Error{
					Location: e.Location,
					Message:  "number of arguments mismatch",
				})
			}
			for i, v := range e.Values {
				eqs = append(eqs, equation{
					left:    v.GetType(),
					right:   tf.Params[i],
					pattern: pattern,
				})
			}

			break
		}
	case *typed.PList:
		{
			e := pattern.(*typed.PList)
			var itemType typed.Type
			for _, item := range e.Items {
				if itemType == nil {
					itemType = item.GetType()
				} else {
					eqs = append(eqs, equation{
						left:    itemType,
						right:   item.GetType(),
						pattern: pattern,
					})
				}
			}
			if itemType == nil {
				itemType = annotateType(nil, nil, e.Location, false)
			}
			eqs = append(eqs, equation{
				left: e.Type,
				right: &typed.TExternal{
					Location: e.Location,
					Name:     common.OakCoreListList,
					Args:     []typed.Type{itemType},
				},
				pattern: pattern,
			})
			for _, item := range e.Items {
				eqs = equatizePattern(eqs, item)
			}

			break
		}
	case *typed.PNamed:
		{
			break
		}
	case *typed.PRecord:
		{
			e := pattern.(*typed.PRecord)
			fields := map[ast.Identifier]typed.Type{}
			for _, f := range e.Fields {
				fields[f.Name] = f.Type
			}

			eqs = append(eqs, equation{
				left: e.Type,
				right: &typed.TRecord{
					Location: e.Location,
					Fields:   fields,
				},
				pattern: pattern,
			})

			break
		}
	case *typed.PTuple:
		{
			e := pattern.(*typed.PTuple)
			eqs = append(eqs, equation{
				left: e.Type,
				right: &typed.TTuple{
					Location: e.Location,
					Items:    common.Map(func(p typed.Pattern) typed.Type { return p.GetType() }, e.Items),
				},
				pattern: pattern,
			})

			for _, item := range e.Items {
				eqs = equatizePattern(eqs, item)
			}
			break
		}
	default:
		panic(common.SystemError{Message: "invalid case"})
	}
	return eqs
}

func equatizeExpression(eqs []equation, expr typed.Expression) []equation {
	if expr == nil {
		return eqs
	}
	switch expr.(type) {
	case *typed.Access:
		{
			e := expr.(*typed.Access)

			fields := map[ast.Identifier]typed.Type{}
			fields[e.FieldName] = e.Type
			eqs = append(eqs, equation{
				left:  e.Record.GetType(),
				right: &typed.TRecord{Location: e.Location, Fields: fields},
				expr:  expr,
			})
			eqs = equatizeExpression(eqs, e.Record)
			break
		}
	case *typed.Call:
		{
			e := expr.(*typed.Call)
			eqs = append(eqs, equation{
				left: e.Func.GetType(),
				right: &typed.TFunc{
					Location: e.Location,
					Params:   common.Map(func(p typed.Expression) typed.Type { return p.GetType() }, e.Args),
					Return:   e.Type,
				},
				expr: expr,
			})
			eqs = equatizeExpression(eqs, e.Func)
			for _, a := range e.Args {
				eqs = equatizeExpression(eqs, a)
			}
			break
		}
	case *typed.Const:
		{
			e := expr.(*typed.Const)
			eqs = append(eqs, equation{
				left:  e.Type,
				right: getConstType(e.Value, e.Location),
				expr:  e,
			})
			break
		}
	case *typed.If:
		{
			e := expr.(*typed.If)
			eqs = append(eqs,
				equation{
					left:  e.Condition.GetType(),
					right: &typed.TExternal{Location: e.Location, Name: common.OakCoreBasicsBool},
					expr:  expr,
				},
				equation{
					left:  e.Type,
					right: e.Positive.GetType(),
					expr:  expr,
				},
				equation{
					left:  e.Type,
					right: e.Negative.GetType(),
					expr:  expr,
				})
			eqs = equatizeExpression(eqs, e.Condition)
			eqs = equatizeExpression(eqs, e.Positive)
			eqs = equatizeExpression(eqs, e.Negative)
			break
		}
	case *typed.Let:
		{
			e := expr.(*typed.Let)
			eqs = append(eqs,
				equation{
					left:  e.Type,
					right: e.Body.GetType(),
					expr:  expr,
				})
			eqs = equatizeExpression(eqs, e.Body)
			break
		}
	case *typed.List:
		{
			e := expr.(*typed.List)
			var listItemType typed.Type

			for _, item := range e.Items {
				if listItemType == nil {
					listItemType = item.GetType()
				} else {
					eqs = append(eqs, equation{
						left:  listItemType,
						right: item.GetType(),
						expr:  expr,
					})
					listItemType = item.GetType()
				}
			}

			if listItemType != nil {
				listItemType = annotateType(nil, nil, e.Location, false)
			}
			eqs = append(eqs, equation{
				left: e.Type,
				right: &typed.TExternal{
					Location: e.Location,
					Name:     common.OakCoreListList,
					Args:     []typed.Type{listItemType},
				},
				expr: expr,
			})

			for _, item := range e.Items {
				eqs = equatizeExpression(eqs, item)
			}
			break
		}
	case *typed.Record:
		{
			e := expr.(*typed.Record)
			fieldTypes := map[ast.Identifier]typed.Type{}
			for _, f := range e.Fields {
				fieldTypes[f.Name] = f.Type
			}

			eqs = append(eqs, equation{
				left: e.Type,
				right: &typed.TRecord{
					Location: e.Location,
					Fields:   fieldTypes,
				},
				expr: expr,
			})

			for _, f := range e.Fields {
				eqs = equatizeExpression(eqs, f.Value)
			}
			break
		}
	case *typed.Select:
		{
			e := expr.(*typed.Select)
			caseType := e.Type
			for _, cs := range e.Cases {
				eqs = append(eqs,
					equation{
						left:  e.Condition.GetType(),
						right: cs.Pattern.GetType(),
						expr:  expr,
					}, equation{
						left:  caseType,
						right: cs.Expression.GetType(),
						expr:  expr,
					})
			}

			for _, cs := range e.Cases {
				eqs = equatizePattern(eqs, cs.Pattern)
				eqs = equatizeExpression(eqs, cs.Expression)
			}
			break
		}
	case *typed.Tuple:
		{
			e := expr.(*typed.Tuple)
			eqs = append(eqs, equation{
				left: e.Type,
				right: &typed.TTuple{
					Location: e.Location,
					Items: common.Map(
						func(e typed.Expression) typed.Type { return e.GetType() },
						e.Items),
				},
				expr: expr,
			})
			for _, item := range e.Items {
				eqs = equatizeExpression(eqs, item)
			}
			break
		}
	case *typed.UpdateLocal:
		{
			e := expr.(*typed.UpdateLocal)
			fieldTypes := map[ast.Identifier]typed.Type{}
			for _, f := range e.Fields {
				fieldTypes[f.Name] = f.Type
			}

			eqs = append(eqs, equation{
				left: e.Type,
				right: &typed.TRecord{
					Location: e.Location,
					Fields:   fieldTypes,
				},
				expr: expr,
			})

			for _, f := range e.Fields {
				eqs = equatizeExpression(eqs, f.Value)
			}
			break
		}
	case *typed.UpdateGlobal:
		{
			e := expr.(*typed.UpdateGlobal)
			fieldTypes := map[ast.Identifier]typed.Type{}
			for _, f := range e.Fields {
				fieldTypes[f.Name] = f.Type
			}

			eqs = append(eqs, equation{
				left: e.Type,
				right: &typed.TRecord{
					Location: e.Location,
					Fields:   fieldTypes,
				},
				expr: expr,
			})

			for _, f := range e.Fields {
				eqs = equatizeExpression(eqs, f.Value)
			}
			equatizeDefinition(eqs, e.Definition)
			break
		}
	case *typed.Lambda:
		{
			e := expr.(*typed.Lambda)
			eqs = append(eqs, equation{
				left: e.Type,
				right: &typed.TFunc{
					Location: e.Location,
					Params:   common.Map(func(p typed.Pattern) typed.Type { return p.GetType() }, e.Params),
					Return:   e.Body.GetType(),
				},
				expr: expr,
			})
			for _, p := range e.Params {
				eqs = equatizePattern(eqs, p)
			}
			eqs = equatizeExpression(eqs, e.Body)
			break
		}
	case *typed.Constructor:
		{
			e := expr.(*typed.Constructor)
			eqs = append(eqs, equation{
				left: e.Type,
				right: &typed.TExternal{
					Location: e.Location,
					Name:     e.DataName,
				},
				expr: e,
			})
			for _, a := range e.Args {
				eqs = equatizeExpression(eqs, a)
			}
		}
	case *typed.NativeCall:
		{
			e := expr.(*typed.NativeCall)
			for _, a := range e.Args {
				eqs = equatizeExpression(eqs, a)
			}
		}
	case *typed.Local:
		{

		}
	case *typed.Global:
		{
			e := expr.(*typed.Global)
			eqs = append(eqs, equation{
				left:  e.Type,
				right: e.Definition.Type,
				expr:  e,
			})
			equatizeDefinition(eqs, e.Definition)
		}
	default:
		panic(common.SystemError{Message: "invalid case"})
	}
	return eqs
}

func annotateDefinition(
	symtab symMap,
	modules map[string]normalized.Module,
	typedModules map[string]*typed.Module,
	def normalized.Definition,
) *typed.Definition {
	o := &typed.Definition{}

	typeParams := typeParamsMap{}
	o.Pattern = annotatePattern(symtab, typeParams, modules, typedModules, def.Pattern)
	o.Type = annotateType(typeParams, def.Type, def.Pattern.(normalized.PNamed).Location, true)
	o.Expression = annotateExpression(symtab, typeParams, modules, typedModules, def.Expression)

	typeMap[o] = o.Type
	return o
}

func annotatePattern(symtab symMap,
	typeParams typeParamsMap,
	modules map[string]normalized.Module,
	typedModules map[string]*typed.Module,
	pattern normalized.Pattern,
) typed.Pattern {
	annotate := func(p normalized.Pattern) typed.Pattern {
		return annotatePattern(symtab, typeParams, modules, typedModules, p)
	}
	var p typed.Pattern
	switch pattern.(type) {
	case normalized.PAlias:
		{
			e := pattern.(normalized.PAlias)
			p = &typed.PAlias{
				Location: e.Location,
				Type:     annotateType(typeParams, e.Type, e.Location, true),
				Alias:    e.Alias,
				Nested:   annotate(e.Nested),
			}
			symtab[e.Alias] = p.GetType()
			break
		}
	case normalized.PAny:
		{
			e := pattern.(normalized.PAny)
			p = &typed.PAny{
				Location: e.Location,
				Type:     annotateType(typeParams, e.Type, e.Location, true),
			}
			break
		}
	case normalized.PCons:
		{
			e := pattern.(normalized.PCons)
			p = &typed.PCons{
				Location: e.Location,
				Type:     annotateType(typeParams, e.Type, e.Location, true),
				Head:     annotate(e.Head),
				Tail:     annotate(e.Tail),
			}
			break
		}
	case normalized.PConst:
		{
			e := pattern.(normalized.PConst)
			p = &typed.PConst{
				Location: e.Location,
				Type:     annotateType(typeParams, e.Type, e.Location, true),
				Value:    e.Value,
			}
			break
		}
	case normalized.PDataValue:
		{
			e := pattern.(normalized.PDataValue)

			t, def := getAnnotatedGlobal(e.ModulePath, e.DefinitionName, modules, typedModules)

			p = &typed.PDataValue{
				Location:       e.Location,
				Type:           t,
				ModulePath:     e.ModulePath,
				DefinitionName: e.DefinitionName,
				Values:         common.Map(annotate, e.Values),
				Definition:     def,
			}
			break
		}
	case normalized.PList:
		{
			e := pattern.(normalized.PList)
			p = &typed.PList{
				Location: e.Location,
				Type:     annotateType(typeParams, e.Type, e.Location, true),
				Items:    common.Map(annotate, e.Items),
			}
			break
		}
	case normalized.PNamed:
		{
			e := pattern.(normalized.PNamed)
			p = &typed.PNamed{
				Location: e.Location,
				Type:     annotateType(typeParams, e.Type, e.Location, true),
				Name:     e.Name,
			}
			symtab[e.Name] = p.GetType()
			break
		}
	case normalized.PRecord:
		{
			e := pattern.(normalized.PRecord)
			p = &typed.PRecord{
				Location: e.Location,
				Type:     annotateType(typeParams, e.Type, e.Location, true),
				Fields: common.Map(func(f normalized.PRecordField) typed.PRecordField {
					return typed.PRecordField{
						Location: f.Location,
						Name:     f.Name,
						Type:     annotateType(typeParams, nil, e.Location, true),
					}
				}, e.Fields),
			}
			break
		}
	case normalized.PTuple:
		{
			e := pattern.(normalized.PTuple)
			p = &typed.PTuple{
				Location: e.Location,
				Type:     annotateType(typeParams, e.Type, e.Location, true),
				Items:    common.Map(annotate, e.Items),
			}
			break
		}
	default:
		panic(common.SystemError{Message: "invalid case"})
	}

	typeMap[p] = p.GetType()
	return p
}

func annotateExpression(
	symtab symMap,
	typeParams typeParamsMap,
	modules map[string]normalized.Module,
	typedModules map[string]*typed.Module,
	expr normalized.Expression,
) typed.Expression {
	if expr == nil {
		return nil
	}

	annotate := func(e normalized.Expression) typed.Expression {
		return annotateExpression(symtab, typeParams, modules, typedModules, e)
	}
	var o typed.Expression
	switch expr.(type) {
	case normalized.Access:
		{
			e := expr.(normalized.Access)
			o = &typed.Access{
				Location:  e.Location,
				Type:      annotateType(typeParams, nil, e.Location, false),
				Record:    annotate(e.Record),
				FieldName: e.FieldName,
			}
			break
		}
	case normalized.Call:
		{
			e := expr.(normalized.Call)
			o = &typed.Call{
				Location: e.Location,
				Type:     annotateType(typeParams, nil, e.Location, false),
				Func:     annotate(e.Func),
				Args:     common.Map(annotate, e.Args),
			}
			break
		}
	case normalized.Const:
		{
			e := expr.(normalized.Const)
			o = &typed.Const{
				Location: e.Location,
				Type:     annotateType(typeParams, nil, e.Location, false),
				Value:    e.Value,
			}
			break
		}
	case normalized.If:
		{
			e := expr.(normalized.If)
			o = &typed.If{
				Location:  e.Location,
				Type:      annotateType(typeParams, nil, e.Location, false),
				Condition: annotate(e.Condition),
				Positive:  annotate(e.Positive),
				Negative:  annotate(e.Negative),
			}
			break
		}
	case normalized.Let:
		{
			e := expr.(normalized.Let)
			o = &typed.Let{
				Location:   e.Location,
				Type:       annotateType(typeParams, nil, e.Location, false),
				Definition: annotateDefinition(symtab, modules, typedModules, e.Definition),
				Body:       annotate(e.Body),
			}
			break
		}
	case normalized.List:
		{
			e := expr.(normalized.List)
			o = &typed.List{
				Location: e.Location,
				Type:     annotateType(typeParams, nil, e.Location, false),
				Items:    common.Map(annotate, e.Items),
			}
			break
		}
	case normalized.Record:
		{
			e := expr.(normalized.Record)
			o = &typed.Record{
				Location: e.Location,
				Type:     annotateType(typeParams, nil, e.Location, false),
				Fields: common.Map(func(f normalized.RecordField) typed.RecordField {
					return typed.RecordField{
						Location: e.Location,
						Type:     annotateType(typeParams, nil, f.Location, false),
						Name:     f.Name,
						Value:    annotate(f.Value),
					}
				}, e.Fields),
			}
			break
		}
	case normalized.Select:
		{
			e := expr.(normalized.Select)
			o = &typed.Select{
				Location:  e.Location,
				Type:      annotateType(typeParams, nil, e.Location, false),
				Condition: annotate(e.Condition),
				Cases: common.Map(func(c normalized.SelectCase) typed.SelectCase {
					return typed.SelectCase{
						Location:   c.Location,
						Type:       annotateType(typeParams, nil, c.Location, false),
						Pattern:    annotatePattern(symtab, typeParams, modules, typedModules, c.Pattern),
						Expression: annotate(c.Expression),
					}
				}, e.Cases),
			}
			break
		}
	case normalized.Tuple:
		{
			e := expr.(normalized.Tuple)
			o = &typed.Tuple{
				Location: e.Location,
				Type:     annotateType(typeParams, nil, e.Location, false),
				Items:    common.Map(annotate, e.Items),
			}
			break
		}
	case normalized.UpdateLocal:
		{
			e := expr.(normalized.UpdateLocal)
			if t, ok := symtab[e.RecordName]; ok {
				o = &typed.UpdateLocal{
					Location:   e.Location,
					Type:       t,
					RecordName: e.RecordName,
					Fields: common.Map(func(f normalized.RecordField) typed.RecordField {
						return typed.RecordField{
							Location: e.Location,
							Type:     annotateType(typeParams, nil, f.Location, false),
							Name:     f.Name,
							Value:    annotate(f.Value),
						}
					}, e.Fields),
				}
			} else {
				panic(common.Error{
					Location: e.Location,
					Message:  "local variable not found",
				})
			}
			break
		}
	case normalized.UpdateGlobal:
		{
			e := expr.(normalized.UpdateGlobal)

			t, def := getAnnotatedGlobal(e.ModulePath, e.DefinitionName, modules, typedModules)

			o = &typed.UpdateGlobal{
				Location:       e.Location,
				Type:           t,
				ModulePath:     e.ModulePath,
				DefinitionName: e.DefinitionName,
				Definition:     def,
				Fields: common.Map(func(f normalized.RecordField) typed.RecordField {
					return typed.RecordField{
						Location: e.Location,
						Type:     annotateType(typeParams, nil, f.Location, false),
						Name:     f.Name,
						Value:    annotate(f.Value),
					}
				}, e.Fields),
			}
			break
		}
	case normalized.Lambda:
		{
			e := expr.(normalized.Lambda)
			localSymtab := symMap{}
			localTypeParams := typeParamsMap{}
			params := common.Map(func(p normalized.Pattern) typed.Pattern {
				return annotatePattern(localSymtab, localTypeParams, modules, typedModules, p)
			}, e.Params)
			mergedSymtab := symMap{}
			maps.Copy(mergedSymtab, symtab)
			maps.Copy(mergedSymtab, localSymtab)

			mergedTypeParams := typeParamsMap{}
			maps.Copy(mergedTypeParams, typeParams)
			maps.Copy(mergedTypeParams, localTypeParams)

			o = &typed.Lambda{
				Location: e.Location,
				Type:     annotateType(mergedTypeParams, nil, e.Location, true),
				Params:   params,
				Body:     annotateExpression(mergedSymtab, mergedTypeParams, modules, typedModules, e.Body),
			}
			break
		}
	case normalized.Constructor:
		{
			e := expr.(normalized.Constructor)
			o = &typed.Constructor{
				Location:  e.Location,
				Type:      annotateType(typeParams, nil, e.Location, false),
				DataName:  e.DataName,
				ValueName: e.ValueName,
				Args:      common.Map(annotate, e.Args),
			}
			break
		}
	case normalized.NativeCall:
		{
			e := expr.(normalized.NativeCall)
			o = &typed.NativeCall{
				Location: e.Location,
				Type:     annotateType(typeParams, nil, e.Location, false),
				Name:     e.Name,
				Args:     common.Map(annotate, e.Args),
			}
			break
		}
	case normalized.Local:
		{
			e := expr.(normalized.Local)
			if t, ok := symtab[e.Name]; ok {
				o = &typed.Local{
					Location: e.Location,
					Type:     t,
					Name:     e.Name,
				}
			} else {
				panic(common.Error{
					Location: e.Location,
					Message:  "local variable not found",
				})
			}
			break
		}
	case normalized.Global:
		{
			e := expr.(normalized.Global)

			t, def := getAnnotatedGlobal(e.ModulePath, e.DefinitionName, modules, typedModules)

			o = &typed.Global{
				Location:       e.Location,
				Type:           t,
				ModulePath:     e.ModulePath,
				DefinitionName: e.DefinitionName,
				Definition:     def,
			}
			break
		}
	default:
		panic(common.SystemError{Message: "invalid case"})
	}

	typeMap[o] = o.GetType()
	return o
}

func getAnnotatedGlobal(
	modulePath string,
	definitionName ast.Identifier,
	modules map[string]normalized.Module,
	typedModules map[string]*typed.Module,
) (typed.Type, *typed.Definition) {
	var t typed.Type
	typedModule, ok := typedModules[modulePath]
	if !ok {
		typedModule = &typed.Module{
			Path:        modulePath,
			Definitions: map[ast.Identifier]*typed.Definition{},
		}
		typedModules[modulePath] = typedModule
	}

	def, ok := typedModule.Definitions[definitionName]
	if !ok {
		defSymtab := symMap{}

		def = annotateDefinition(
			defSymtab, modules, typedModules, modules[modulePath].Definitions[definitionName])
		typedModule.Definitions[definitionName] = def
	}

	t = def.Type
	return t, def
}

func annotateType(typeParams typeParamsMap, t normalized.Type, location ast.Location, typeMapSource bool) typed.Type {
	annotate := func(l ast.Location) func(x normalized.Type) typed.Type {
		return func(x normalized.Type) typed.Type {
			return annotateType(typeParams, x, location, typeMapSource)
		}
	}

	var r typed.Type
	if t == nil {
		unboundIndex++
		r = &typed.TUnbound{
			Location: location,
			Index:    unboundIndex,
		}
	} else {

		switch t.(type) {
		case normalized.TFunc:
			{
				e := t.(normalized.TFunc)
				r = &typed.TFunc{
					Location: e.Location,
					Params:   common.Map(annotate(e.Location), e.Params),
					Return:   annotateType(typeParams, e.Return, e.Location, typeMapSource),
				}
				break
			}
		case normalized.TRecord:
			{
				e := t.(normalized.TRecord)
				fields := map[ast.Identifier]typed.Type{}
				for n, v := range e.Fields {
					fields[n] = annotateType(typeParams, v, e.Location, typeMapSource)
				}
				r = &typed.TRecord{
					Location: e.Location,
					Fields:   fields,
				}
				break
			}
		case normalized.TTuple:
			{
				e := t.(normalized.TTuple)
				r = &typed.TTuple{
					Location: e.Location,
					Items:    common.Map(annotate(e.Location), e.Items),
				}
				break
			}
		case normalized.TUnit:
			{
				e := t.(normalized.TUnit)
				r = &typed.TExternal{Location: e.Location, Name: common.OakCoreBasicsUnit}
				break
			}
		case normalized.TData:
			{
				e := t.(normalized.TData)
				r = &typed.TExternal{
					Location: e.Location,
					Name:     e.Name,
					Args:     common.Map(annotate(e.Location), e.Args),
				}
				break
			}
		case normalized.TExternal:
			{
				e := t.(normalized.TExternal)
				r = &typed.TExternal{
					Location: e.Location,
					Name:     e.Name,
					Args:     common.Map(annotate(e.Location), e.Args),
				}

				break
			}
		case normalized.TTypeParameter:
			{
				e := t.(normalized.TTypeParameter)
				//TODO: constraints
				if typeMapSource {
					r = annotateType(typeParams, nil, e.Location, true)
					typeParams[e.Name] = r.(*typed.TUnbound).Index
				} else {
					if id, ok := typeParams[e.Name]; ok {
						r = &typed.TUnbound{
							Location: e.Location,
							Index:    id,
						}
					} else {
						panic(common.Error{Location: e.Location, Message: "unknown type parameter"})
					}
				}
				break
			}
		default:
			panic(common.SystemError{Message: "invalid case"})

		}
	}
	return r
}

func getConstType(cv ast.ConstValue, location ast.Location) typed.Type {
	switch cv.(type) {
	case ast.CChar:
		return &typed.TExternal{Location: location, Name: common.OakCoreCharChar}
	case ast.CInt:
		return &typed.TExternal{Location: location, Name: common.OakCoreBasicsInt}
	case ast.CFloat:
		return &typed.TExternal{Location: location, Name: common.OakCoreBasicsFloat}
	case ast.CString:
		return &typed.TExternal{Location: location, Name: common.OakCoreStringString}
	case ast.CUnit:
		return &typed.TExternal{Location: location, Name: common.OakCoreBasicsUnit}
	}
	panic(common.SystemError{Message: "invalid case"})
}

func unifyAll(eqs []equation) map[uint64]typed.Type {
	subst := map[uint64]typed.Type{}
	for _, eq := range eqs {
		loc := eq.left.GetLocation()
		if eq.expr != nil {
			loc = eq.expr.GetLocation()
		}
		if eq.pattern != nil {
			loc = eq.pattern.GetLocation()
		}
		if eq.def != nil {
			loc = eq.def.Expression.GetLocation()
		}
		unify(eq.left, eq.right, loc, subst)
	}
	return subst
}

func balanceFn(f *typed.TFunc, sz int) *typed.TFunc {
	if len(f.Params) == sz {
		return f
	}

	return &typed.TFunc{
		Location: f.Location,
		Params:   f.Params[0:sz],
		Return: &typed.TFunc{
			Location: f.Location,
			Params:   f.Params[sz:],
			Return:   f.Return,
		},
	}
}

func unify(x typed.Type, y typed.Type, loc ast.Location, subst map[uint64]typed.Type) {
	if x.EqualsTo(y) {
		return
	}
	if _, ub := x.(*typed.TUnbound); ub {
		unifyUnbound(x.(*typed.TUnbound), y, loc, subst)
		return
	}
	if _, ub := y.(*typed.TUnbound); ub {
		unifyUnbound(y.(*typed.TUnbound), x, loc, subst)
		return
	}
	switch x.(type) {
	case *typed.TFunc:
		{
			if ey, ok := y.(*typed.TFunc); ok {
				ex := x.(*typed.TFunc)
				if len(ex.Params) < len(ey.Params) {
					panic(common.Error{Location: ex.Location, Message: "func parameters mismatch"})
				}
				ex = balanceFn(ex, len(ey.Params))
				for i, p := range ex.Params {
					unify(p, ey.Params[i], loc, subst)
				}
				unify(ex.Return, ey.Return, loc, subst)
				return
			}
			break
		}
	case *typed.TRecord:
		{
			if ey, ok := y.(*typed.TRecord); ok {
				ex := x.(*typed.TRecord)
				if len(ex.Fields) != len(ey.Fields) {
					//TODO: prefer intersection match?
					panic(common.Error{Location: ex.Location, Message: "record fields number mismatch"})
				}
				for i, f := range ex.Fields {
					unify(f, ey.Fields[i], loc, subst)
				}
				return
			}
			break
		}
	case *typed.TTuple:
		{
			if ey, ok := y.(*typed.TTuple); ok {
				ex := x.(*typed.TTuple)
				if len(ex.Items) != len(ey.Items) {
					panic(common.Error{Location: ex.Location, Message: "tuple lengths mismatch"})
				}
				for i, p := range ex.Items {
					unify(p, ey.Items[i], loc, subst)
				}
				return
			}
			break
		}
	case *typed.TExternal:
		{
			if ey, ok := y.(*typed.TExternal); ok {
				ex := x.(*typed.TExternal)
				if ex.Name == ey.Name {
					if len(ex.Args) == len(ey.Args) {
						for i, p := range ex.Args {
							unify(p, ey.Args[i], loc, subst)
						}
					}
					return
				}
			}
			break
		}
	default:
		panic(common.SystemError{Message: "invalid case"})
	}
	panic(common.Error{Location: loc, Message: fmt.Sprintf("type %v and %v do not match", x, y)})
}

func unifyUnbound(v *typed.TUnbound, typ typed.Type, loc ast.Location, subst map[uint64]typed.Type) {
	if x, ok := subst[v.Index]; ok {
		unify(x, typ, loc, subst)
	} else {
		if y, ok := typ.(*typed.TUnbound); ok {
			if _, c := subst[y.Index]; c {
				unify(v, subst[y.Index], loc, subst)
			}
		}
		if v.OccursCheck(typ, subst) {
			panic(common.Error{Location: v.GetLocation(), Message: fmt.Sprintf("ambigous type: %v vs %v", v, typ)})
		}
	}
	subst[v.Index] = typ
}

func applyDefinition(td *typed.Definition, subst map[uint64]typed.Type) *typed.Definition {
	td.Type = applyType(td.Type, subst)
	td.Pattern = applyPattern(td.Pattern, subst)
	td.Expression = applyExpression(td.Expression, subst)
	return td
}

func applyType(t typed.Type, subst map[uint64]typed.Type) typed.Type {
	apply := func(x typed.Type) typed.Type {
		return applyType(x, subst)
	}

	switch t.(type) {
	case *typed.TFunc:
		{
			e := t.(*typed.TFunc)
			t = &typed.TFunc{
				Location: e.Location,
				Params:   common.Map(apply, e.Params),
				Return:   applyType(e.Return, subst),
			}
			break
		}
	case *typed.TRecord:
		{
			e := t.(*typed.TRecord)
			var fields map[ast.Identifier]typed.Type
			for n, x := range e.Fields {
				fields[n] = apply(x)
			}
			t = &typed.TRecord{
				Location: e.Location,
				Fields:   fields,
			}
			break
		}
	case *typed.TTuple:
		{
			e := t.(*typed.TTuple)
			t = &typed.TTuple{
				Location: e.Location,
				Items:    common.Map(apply, e.Items),
			}
			break
		}
	case *typed.TExternal:
		{
			e := t.(*typed.TExternal)
			t = &typed.TExternal{
				Location: e.Location,
				Name:     e.Name,
				Args:     common.Map(apply, e.Args),
			}
			break
		}
	case *typed.TUnbound:
		{
			e := t.(*typed.TUnbound)
			if x, ok := subst[e.Index]; ok {
				t = apply(x)
			}
			break
		}
	default:
		panic(common.SystemError{Message: "invalid case"})
	}
	return t
}

func applyPattern(pattern typed.Pattern, subst map[uint64]typed.Type) typed.Pattern {
	apply := func(x typed.Pattern) typed.Pattern {
		return applyPattern(x, subst)
	}
	switch pattern.(type) {
	case *typed.PAlias:
		{
			e := pattern.(*typed.PAlias)
			pattern = &typed.PAlias{
				Location: e.Location,
				Type:     applyType(e.Type, subst),
				Alias:    e.Alias,
				Nested:   apply(e.Nested),
			}
			break
		}
	case *typed.PAny:
		{
			e := pattern.(*typed.PAny)
			pattern = &typed.PAny{
				Location: e.Location,
				Type:     applyType(e.Type, subst),
			}
			break
		}
	case *typed.PCons:
		{
			e := pattern.(*typed.PCons)
			pattern = &typed.PCons{
				Location: e.Location,
				Type:     applyType(e.Type, subst),
				Head:     apply(e.Head),
				Tail:     apply(e.Head),
			}
			break
		}
	case *typed.PConst:
		{
			e := pattern.(*typed.PConst)
			pattern = &typed.PConst{
				Location: e.Location,
				Type:     applyType(e.Type, subst),
				Value:    e.Value,
			}
			break
		}
	case *typed.PDataValue:
		{
			e := pattern.(*typed.PDataValue)
			pattern = &typed.PDataValue{
				Location:       e.Location,
				Type:           applyType(e.Type, subst),
				ModulePath:     e.ModulePath,
				DefinitionName: e.DefinitionName,
				Definition:     e.Definition,
				Values:         common.Map(apply, e.Values),
			}
			break
		}
	case *typed.PList:
		{
			e := pattern.(*typed.PList)
			pattern = &typed.PList{
				Location: e.Location,
				Type:     applyType(e.Type, subst),
				Items:    common.Map(apply, e.Items),
			}
			break
		}
	case *typed.PNamed:
		{
			e := pattern.(*typed.PNamed)
			pattern = &typed.PNamed{
				Location: ast.Location{},
				Type:     applyType(e.Type, subst),
				Name:     e.Name,
			}
			break
		}
	case *typed.PRecord:
		{
			e := pattern.(*typed.PRecord)
			pattern = &typed.PRecord{
				Location: e.Location,
				Type:     applyType(e.Type, subst),
				Fields: common.Map(func(f typed.PRecordField) typed.PRecordField {
					return typed.PRecordField{
						Location: f.Location,
						Name:     f.Name,
						Type:     applyType(f.Type, subst),
					}
				}, e.Fields),
			}
			break
		}
	case *typed.PTuple:
		{
			e := pattern.(*typed.PTuple)
			pattern = &typed.PTuple{
				Location: e.Location,
				Type:     applyType(e.Type, subst),
				Items:    common.Map(apply, e.Items),
			}
			break
		}
	default:
		panic(common.SystemError{Message: "invalid case"})
	}
	return pattern
}

func applyExpression(expr typed.Expression, subst map[uint64]typed.Type) typed.Expression {
	if expr == nil {
		return nil
	}

	apply := func(x typed.Expression) typed.Expression {
		return applyExpression(x, subst)
	}
	switch expr.(type) {
	case *typed.Access:
		{
			e := expr.(*typed.Access)
			expr = &typed.Access{
				Location:  e.Location,
				Type:      applyType(e.Type, subst),
				FieldName: e.FieldName,
				Record:    apply(e.Record),
			}
			break
		}
	case *typed.Call:
		{
			e := expr.(*typed.Call)
			expr = &typed.Call{
				Location: e.Location,
				Type:     applyType(e.Type, subst),
				Func:     apply(e.Func),
				Args:     common.Map(apply, e.Args),
			}
			break
		}
	case *typed.Const:
		{
			e := expr.(*typed.Const)
			expr = &typed.Const{
				Location: e.Location,
				Type:     applyType(e.Type, subst),
				Value:    e.Value,
			}
			break
		}
	case *typed.If:
		{
			e := expr.(*typed.If)
			expr = &typed.If{
				Location:  e.Location,
				Type:      applyType(e.Type, subst),
				Condition: apply(e.Condition),
				Positive:  apply(e.Positive),
				Negative:  apply(e.Negative),
			}
			break
		}
	case *typed.Let:
		{
			e := expr.(*typed.Let)
			expr = &typed.Let{
				Location:   e.Location,
				Type:       applyType(e.Type, subst),
				Definition: applyDefinition(e.Definition, subst),
				Body:       apply(e.Body),
			}
			break
		}
	case *typed.List:
		{
			e := expr.(*typed.List)
			expr = &typed.List{
				Location: e.Location,
				Type:     applyType(e.Type, subst),
				Items:    common.Map(apply, e.Items),
			}
			break
		}
	case *typed.Record:
		{
			e := expr.(*typed.Record)
			expr = &typed.Record{
				Location: e.Location,
				Type:     applyType(e.Type, subst),
				Fields: common.Map(func(x typed.RecordField) typed.RecordField {
					return typed.RecordField{
						Location: x.Location,
						Type:     applyType(x.Type, subst),
						Name:     x.Name,
						Value:    apply(x.Value),
					}
				}, e.Fields),
			}
			break
		}
	case *typed.Select:
		{
			e := expr.(*typed.Select)
			expr = &typed.Select{
				Location:  e.Location,
				Type:      applyType(e.Type, subst),
				Condition: apply(e.Condition),
				Cases: common.Map(func(x typed.SelectCase) typed.SelectCase {
					return typed.SelectCase{
						Location:   x.Location,
						Type:       applyType(x.Type, subst),
						Pattern:    applyPattern(x.Pattern, subst),
						Expression: apply(x.Expression),
					}
				}, e.Cases),
			}
			break
		}
	case *typed.Tuple:
		{
			e := expr.(*typed.Tuple)

			expr = &typed.Tuple{
				Location: e.Location,
				Type:     applyType(e.Type, subst),
				Items:    common.Map(apply, e.Items),
			}

			break
		}
	case *typed.UpdateLocal:
		{
			e := expr.(*typed.UpdateLocal)
			expr = &typed.UpdateLocal{
				Location:   e.Location,
				Type:       applyType(e.Type, subst),
				RecordName: e.RecordName,
				Fields: common.Map(func(x typed.RecordField) typed.RecordField {
					return typed.RecordField{
						Location: x.Location,
						Type:     applyType(x.Type, subst),
						Name:     x.Name,
						Value:    apply(x.Value),
					}
				}, e.Fields),
			}
			break
		}
	case *typed.
		UpdateGlobal:
		{
			e := expr.(*typed.UpdateGlobal)
			expr = &typed.UpdateGlobal{
				Location:       e.Location,
				Type:           applyType(e.Type, subst),
				ModulePath:     e.ModulePath,
				DefinitionName: e.DefinitionName,
				Definition:     e.Definition,
				Fields: common.Map(func(x typed.RecordField) typed.RecordField {
					return typed.RecordField{
						Location: x.Location,
						Type:     applyType(x.Type, subst),
						Name:     x.Name,
						Value:    apply(x.Value),
					}
				}, e.Fields),
			}
			break
		}
	case *typed.Lambda:
		{
			e := expr.(*typed.Lambda)
			expr = &typed.Lambda{
				Location: e.Location,
				Type:     applyType(e.Type, subst),
				Params: common.Map(func(p typed.Pattern) typed.Pattern {
					return applyPattern(p, subst)
				}, e.Params),
				Body: apply(e.Body),
			}
			break
		}
	case *typed.Constructor:
		{
			e := expr.(*typed.Constructor)
			expr = &typed.Constructor{
				Location:  e.Location,
				Type:      applyType(e.Type, subst),
				DataName:  e.DataName,
				ValueName: e.ValueName,
				Args:      common.Map(apply, e.Args),
			}

			break
		}
	case *typed.NativeCall:
		{
			e := expr.(*typed.NativeCall)
			expr = &typed.NativeCall{
				Location: e.Location,
				Type:     applyType(e.Type, subst),
				Name:     e.Name,
				Args:     common.Map(apply, e.Args),
			}
			break
		}

	case *typed.Local:
		{
			e := expr.(*typed.Local)
			expr = &typed.Local{
				Location: e.Location,
				Type:     applyType(e.Type, subst),
				Name:     e.Name,
			}
			break
		}
	case *typed.Global:
		{
			e := expr.(*typed.Global)
			expr = &typed.Global{
				Location:       e.Location,
				Type:           applyType(e.Type, subst),
				ModulePath:     e.ModulePath,
				DefinitionName: e.DefinitionName,
				Definition:     e.Definition,
			}
			break
		}
	default:
		panic(common.SystemError{Message: "invalid case"})
	}
	return expr
}

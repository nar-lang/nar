package processors

import (
	"fmt"
	"maps"
	"oak-compiler/internal/pkg/ast"
	"oak-compiler/internal/pkg/ast/normalized"
	"oak-compiler/internal/pkg/ast/typed"
	"oak-compiler/internal/pkg/common"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var unboundIndex = uint64(0)
var annotations []struct {
	fmt.Stringer
	typed.Type
}

const dumpDebugOutput = false

type symbolsMap map[ast.Identifier]typed.Type
type localDefsMap map[ast.Identifier]*typed.Definition
type typeParamsMap map[ast.Identifier]uint64

type equation struct {
	left, right typed.Type
	loc         *ast.Location
	expr        typed.Expression
	pattern     typed.Pattern
	def         *typed.Definition
}

func (e equation) String(index int) string {
	x := fmt.Stringer(e.expr)
	if x == nil {
		x = e.pattern
	}
	if x == nil {
		x = e.def
	}
	return fmt.Sprintf("\n| %d | `%v` | `%v` | `%v` |", index, e.left, e.right, x)
}

func Solve(
	moduleName ast.QualifiedIdentifier,
	modules map[ast.QualifiedIdentifier]*normalized.Module,
	typedModules map[ast.QualifiedIdentifier]*typed.Module,
) {
	if _, ok := typedModules[moduleName]; ok {
		return
	}

	m := modules[moduleName]
	for _, dep := range m.Dependencies {
		Solve(dep, modules, typedModules)
	}

	o := typed.Module{
		Name:         m.Name,
		Dependencies: m.Dependencies,
	}

	typedModules[o.Name] = &o

	for i := 0; i < len(m.Definitions); i++ {
		def := m.Definitions[i]
		if slices.ContainsFunc(o.Definitions, func(d *typed.Definition) bool { return d.Id == def.Id }) {
			continue
		}

		fp := fmt.Sprintf(".oak-bin/%s/%s.md", m.Name, def.Name)
		sb := strings.Builder{}

		unboundIndex = 0
		annotations = []struct {
			fmt.Stringer
			typed.Type
		}{}
		localTyped := map[ast.QualifiedIdentifier]*typed.Module{}
		var eqs []equation

		td, _, _ := annotateDefinition(symbolsMap{}, typeParamsMap{}, modules, localTyped, m.Name, def, nil)

		if dumpDebugOutput {
			_ = os.MkdirAll(filepath.Dir(fp), os.ModePerm)
			sb.WriteString(fmt.Sprintf("\n\nDefinition\n---\n`%s`", td))
			sb.WriteString("\n\nAnnotations\n---\n| Node | Type |\n|---|---|")
			for _, t := range annotations {
				sb.WriteString(fmt.Sprintf("\n| `%v` | `%v` |", t.Stringer, t.Type))
			}
			_ = os.WriteFile(fp, []byte(sb.String()), 0666)
		}

		eqs = equatizeDefinition(eqs, td, localDefsMap{}, nil, nil)

		if dumpDebugOutput {
			sb.WriteString("\n\nEquations\n---\n| No | Left | Right | Node |\n|---|---|---|---|")
			for i, eq := range eqs {
				sb.WriteString(eq.String(i))
			}
			_ = os.WriteFile(fp, []byte(sb.String()), 0666)
		}

		subst := unifyAll(eqs, []ast.Location{def.Location})

		if dumpDebugOutput {
			sb.WriteString("\n\nUnified\n---\n| Left | Right |\n|---|---|")
			for k, v := range subst {
				sb.WriteString(fmt.Sprintf("\n | `%v` | `%v` |", &typed.TUnbound{Index: k}, v))
			}
			_ = os.WriteFile(fp, []byte(sb.String()), 0666)
		}

		td = applyDefinition(td, subst)

		if dumpDebugOutput {
			sb.WriteString("\n\nSolved\n---\n")
			sb.WriteString(fmt.Sprintf("\n `%v`", td.GetType()))
			_ = os.WriteFile(fp, []byte(sb.String()), 0666)
		}

		o.Definitions = append(o.Definitions, td)
	}
}

func annotateDefinition(
	symbols symbolsMap,
	typeParams typeParamsMap,
	modules map[ast.QualifiedIdentifier]*normalized.Module,
	typedModules map[ast.QualifiedIdentifier]*typed.Module,
	moduleName ast.QualifiedIdentifier,
	def *normalized.Definition,
	stack []*typed.Definition,
) (*typed.Definition, symbolsMap, typeParamsMap) {
	o := &typed.Definition{
		Id:       def.Id,
		Name:     def.Name,
		Hidden:   def.Hidden,
		Location: def.Location,
	}

	localSymbols := symbolsMap{}
	localTypeParams := typeParamsMap{}

	o.Params = common.Map(func(p normalized.Pattern) typed.Pattern {
		return annotatePattern(localSymbols, localTypeParams, modules, typedModules, moduleName, p, true, stack)
	}, def.Params)

	mergedSymbols := symbolsMap{}
	maps.Copy(mergedSymbols, symbols)
	maps.Copy(mergedSymbols, localSymbols)

	mergedTypeParams := typeParamsMap{}
	maps.Copy(mergedTypeParams, typeParams)
	maps.Copy(mergedTypeParams, localTypeParams)

	if def.Type != nil {
		o.DefinedType = annotateType(mergedTypeParams, def.Type, def.Location, true)
	}

	for _, std := range stack {
		if std.Id == def.Id {
			return std, mergedSymbols, mergedTypeParams
		}
	}

	stack = append(stack, o)
	o.Expression = annotateExpression(mergedSymbols, mergedTypeParams, modules, typedModules, moduleName, def.Expression, stack)
	stack = stack[:len(stack)-1]
	return o, mergedSymbols, mergedTypeParams
}

func annotatePattern(symbols symbolsMap,
	typeParams typeParamsMap,
	modules map[ast.QualifiedIdentifier]*normalized.Module,
	typedModules map[ast.QualifiedIdentifier]*typed.Module,
	moduleName ast.QualifiedIdentifier,
	pattern normalized.Pattern,
	typeMapSource bool,
	stack []*typed.Definition,
) typed.Pattern {
	if pattern == nil {
		return nil
	}
	annotate := func(p normalized.Pattern) typed.Pattern {
		return annotatePattern(symbols, typeParams, modules, typedModules, moduleName, p, typeMapSource, stack)
	}
	var p typed.Pattern
	switch pattern.(type) {
	case normalized.PAlias:
		{
			e := pattern.(normalized.PAlias)
			p = &typed.PAlias{
				Location: e.Location,
				Type:     annotateType(typeParams, e.Type, e.Location, typeMapSource),
				Alias:    e.Alias,
				Nested:   annotate(e.Nested),
			}
			symbols[e.Alias] = p.GetType()
			break
		}
	case normalized.PAny:
		{
			e := pattern.(normalized.PAny)
			p = &typed.PAny{
				Location: e.Location,
				Type:     annotateType(typeParams, e.Type, e.Location, typeMapSource),
			}
			break
		}
	case normalized.PCons:
		{
			e := pattern.(normalized.PCons)
			p = &typed.PCons{
				Location: e.Location,
				Type:     annotateType(typeParams, e.Type, e.Location, typeMapSource),
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
				Type:     annotateType(typeParams, e.Type, e.Location, typeMapSource),
				Value:    e.Value,
			}
			break
		}
	case normalized.PDataOption:
		{
			e := pattern.(normalized.PDataOption)
			def := getAnnotatedGlobal(e.ModuleName, e.DefinitionName, modules, typedModules, stack)
			if ctor, ok := def.Expression.(*typed.Constructor); !ok {
				panic(common.SystemError{Message: "data option definition is not a constructor"})
			} else {
				p = &typed.PDataOption{
					Location:   e.Location,
					Type:       annotateType(typeParams, nil, e.Location, typeMapSource),
					Name:       ctor.OptionName,
					Args:       common.Map(annotate, e.Values),
					Definition: def,
				}
			}
			break
		}
	case normalized.PList:
		{
			e := pattern.(normalized.PList)
			p = &typed.PList{
				Location: e.Location,
				Type:     annotateType(typeParams, e.Type, e.Location, typeMapSource),
				Items:    common.Map(annotate, e.Items),
			}
			break
		}
	case normalized.PNamed:
		{
			e := pattern.(normalized.PNamed)
			p = &typed.PNamed{
				Location: e.Location,
				Type:     annotateType(typeParams, nil, e.Location, typeMapSource),
				Name:     e.Name,
			}
			symbols[e.Name] = p.GetType()
			break
		}
	case normalized.PRecord:
		{
			e := pattern.(normalized.PRecord)
			p = &typed.PRecord{
				Location: e.Location,
				Type:     annotateType(typeParams, e.Type, e.Location, typeMapSource),
				Fields: common.Map(func(f normalized.PRecordField) typed.PRecordField {
					return typed.PRecordField{
						Location: f.Location,
						Name:     f.Name,
						Type:     annotateType(typeParams, nil, e.Location, typeMapSource),
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
				Type:     annotateType(typeParams, e.Type, e.Location, typeMapSource),
				Items:    common.Map(annotate, e.Items),
			}
			break
		}
	default:
		panic(common.SystemError{Message: "invalid case"})
	}

	annotations = append(annotations, struct {
		fmt.Stringer
		typed.Type
	}{p, p.GetType()})
	return p
}

func annotateExpression(
	symbols symbolsMap,
	typeParams typeParamsMap,
	modules map[ast.QualifiedIdentifier]*normalized.Module,
	typedModules map[ast.QualifiedIdentifier]*typed.Module,
	moduleName ast.QualifiedIdentifier,
	expr normalized.Expression,
	stack []*typed.Definition,
) typed.Expression {
	if expr == nil {
		return nil
	}

	annotate := func(e normalized.Expression) typed.Expression {
		return annotateExpression(symbols, typeParams, modules, typedModules, moduleName, e, stack)
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
	case normalized.Apply:
		{
			e := expr.(normalized.Apply)
			o = &typed.Apply{
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
	case normalized.LetMatch:
		{
			e := expr.(normalized.LetMatch)

			localSymbols := maps.Clone(symbols)
			localTypeParams := maps.Clone(typeParams)

			o = &typed.Let{
				Location: e.Location,
				Type:     annotateType(localTypeParams, nil, e.Location, true),
				Pattern:  annotatePattern(localSymbols, localTypeParams, modules, typedModules, moduleName, e.Pattern, true, stack),
				Value:    annotateExpression(localSymbols, localTypeParams, modules, typedModules, moduleName, e.Value, stack),
				Body:     annotateExpression(localSymbols, localTypeParams, modules, typedModules, moduleName, e.Nested, stack),
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
					localSymbols := maps.Clone(symbols)
					localTypeParams := maps.Clone(typeParams)
					return typed.SelectCase{
						Location: c.Location,
						Pattern: annotatePattern(
							localSymbols, localTypeParams, modules, typedModules, moduleName, c.Pattern, false, stack),
						Expression: annotateExpression(
							localSymbols, localTypeParams, modules, typedModules, moduleName, c.Expression, stack),
						Type: annotateType(localTypeParams, nil, c.Location, false),
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
			if t, ok := symbols[e.RecordName]; ok {
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
					Message:  fmt.Sprintf("local variable `%s` not found", e.RecordName),
				})
			}
			break
		}
	case normalized.UpdateGlobal:
		{
			e := expr.(normalized.UpdateGlobal)

			def := getAnnotatedGlobal(e.ModuleName, e.DefinitionName, modules, typedModules, stack)

			o = &typed.UpdateGlobal{
				Location:       e.Location,
				Type:           def.GetType(),
				ModuleName:     e.ModuleName,
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
	case normalized.Constructor:
		{
			e := expr.(normalized.Constructor)
			o = &typed.Constructor{
				Location:   e.Location,
				Type:       annotateType(typeParams, nil, e.Location, false),
				DataName:   e.DataName,
				OptionName: common.MakeDataOptionIdentifier(e.DataName, e.OptionName),
				Args:       common.Map(annotate, e.Args),
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
			localType, ok := symbols[e.Name]
			if !ok {
				panic(common.Error{
					Location: e.Location, Message: fmt.Sprintf("local variable `%s` not found", e.Name),
				})
			}
			o = &typed.Local{
				Location: e.Location,
				Type:     localType,
				Name:     e.Name,
			}
			break
		}
	case normalized.Global:
		{
			e := expr.(normalized.Global)
			def := getAnnotatedGlobal(e.ModuleName, e.DefinitionName, modules, typedModules, stack)

			dt := def.GetType()
			if dt == nil {
				dt = annotateType(typeParams, nil, e.Location, false)
			}
			o = &typed.Global{
				Location:       e.Location,
				Type:           dt,
				ModuleName:     e.ModuleName,
				DefinitionName: e.DefinitionName,
				Definition:     def,
			}
			break
		}
	default:
		panic(common.SystemError{Message: "invalid case"})
	}

	annotations = append(annotations, struct {
		fmt.Stringer
		typed.Type
	}{o, o.GetType()})
	return o
}

func getAnnotatedGlobal(
	moduleName ast.QualifiedIdentifier,
	definitionName ast.Identifier,
	modules map[ast.QualifiedIdentifier]*normalized.Module,
	typedModules map[ast.QualifiedIdentifier]*typed.Module,
	stack []*typed.Definition,
) *typed.Definition {
	nDef, ok := common.Find(func(definition *normalized.Definition) bool {
		return definition.Name == definitionName
	}, modules[moduleName].Definitions)
	if !ok {
		panic(common.SystemError{
			Message: fmt.Sprintf("definition `%s` not found", definitionName),
		})
	}

	def, ok := common.Find(func(definition *typed.Definition) bool {
		return definition.Id == nDef.Id
	}, stack)

	if !ok {
		def, _, _ = annotateDefinition(symbolsMap{}, typeParamsMap{}, modules, typedModules, moduleName, nDef, stack)
	}

	return def
}

func newAnnotatedType(loc ast.Location, constraint typed.Constraint) *typed.TUnbound {
	unboundIndex++
	return &typed.TUnbound{
		Location:   loc,
		Index:      unboundIndex,
		Constraint: constraint,
	}
}

func annotateType(
	typeParams typeParamsMap, t normalized.Type, location ast.Location, typeMapSource bool,
) typed.Type {
	annotate := func(l ast.Location) func(x normalized.Type) typed.Type {
		return func(x normalized.Type) typed.Type {
			return annotateType(typeParams, x, location, typeMapSource)
		}
	}

	var r typed.Type
	if t == nil {
		r = newAnnotatedType(location, typed.ConstraintNone)
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

				constraint := typed.ConstraintNone
				if strings.HasPrefix(string(e.Name), string(typed.ConstraintNumber)) {
					constraint = typed.ConstraintNumber
				}
				if strings.HasPrefix(string(e.Name), string(typed.ConstraintComparable)) {
					constraint = typed.ConstraintComparable
				}

				if id, ok := typeParams[e.Name]; ok {
					r = &typed.TUnbound{
						Location:   e.Location,
						Index:      id,
						Constraint: constraint,
					}
				} else {
					if typeMapSource {
						r = annotateType(typeParams, nil, e.Location, true)
						annotations = append(annotations, struct {
							fmt.Stringer
							typed.Type
						}{e, r})
						typeParams[e.Name] = r.(*typed.TUnbound).Index
					} else {
						panic(common.Error{
							Location: e.Location, Message: "unknown type parameter",
						})
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

func equatizeDefinition(
	eqs []equation, td *typed.Definition, localDefs localDefsMap, stack []*typed.Definition, loc *ast.Location,
) []equation {
	for _, std := range stack {
		if std.Id == td.Id {
			return eqs
		}
	}
	stack = append(stack, td)

	if td.Expression != nil && td.DefinedType != nil {
		defType := td.Expression.GetType()

		if len(td.Params) > 0 {
			defType = &typed.TFunc{
				Location: td.Location,
				Params:   common.Map(func(x typed.Pattern) typed.Type { return x.GetType() }, td.Params),
				Return:   defType,
			}
		}

		eqs = append(eqs, equation{
			loc:   loc,
			left:  td.DefinedType,
			right: defType,
			def:   td,
		})
	}

	for _, p := range td.Params {
		eqs = equatizePattern(eqs, p, loc)
	}

	if td.Expression != nil {
		eqs = equatizeExpression(eqs, td.Expression, localDefs, stack, loc)
	}

	stack = stack[:len(stack)-1]
	return eqs
}

func equatizePattern(eqs []equation, pattern typed.Pattern, loc *ast.Location) []equation {
	switch pattern.(type) {
	case *typed.PAlias:
		{
			e := pattern.(*typed.PAlias)
			eqs = append(eqs,
				equation{
					loc:     loc,
					left:    e.Type,
					right:   e.Nested.GetType(),
					pattern: pattern,
				})
			eqs = equatizePattern(eqs, e.Nested, loc)
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
					loc:     loc,
					left:    e.Type,
					right:   e.Tail.GetType(),
					pattern: pattern,
				},
				equation{
					loc:  loc,
					left: e.Tail.GetType(),
					right: &typed.TExternal{
						Location: e.Location,
						Name:     common.OakCoreListList,
						Args:     []typed.Type{e.Head.GetType()},
					},
					pattern: pattern,
				})
			eqs = equatizePattern(eqs, e.Head, loc)
			eqs = equatizePattern(eqs, e.Tail, loc)
			break
		}
	case *typed.PConst:
		{
			e := pattern.(*typed.PConst)
			eqs = append(eqs, equation{
				loc:     loc,
				left:    e.Type,
				right:   getConstType(e.Value, e.Location),
				pattern: pattern,
			})
			break
		}
	case *typed.PDataOption:
		{
			e := pattern.(*typed.PDataOption)
			if len(e.Args) == 0 {
				eqs = append(eqs, equation{
					loc:     loc,
					left:    e.Type,
					right:   e.Definition.GetType(),
					pattern: pattern,
				})
			} else {
				eqs = append(eqs, equation{
					loc: loc,
					left: &typed.TFunc{
						Location: e.Location,
						Params:   common.Map(func(x typed.Pattern) typed.Type { return x.GetType() }, e.Args),
						Return:   e.Type,
					},
					right:   e.Definition.GetType(),
					pattern: e,
				})
				for _, arg := range e.Args {
					eqs = equatizePattern(eqs, arg, loc)
				}
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
						loc:     loc,
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
				loc:  loc,
				left: e.Type,
				right: &typed.TExternal{
					Location: e.Location,
					Name:     common.OakCoreListList,
					Args:     []typed.Type{itemType},
				},
				pattern: pattern,
			})
			for _, item := range e.Items {
				eqs = equatizePattern(eqs, item, loc)
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
				loc:  loc,
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
				loc:  loc,
				left: e.Type,
				right: &typed.TTuple{
					Location: e.Location,
					Items:    common.Map(func(p typed.Pattern) typed.Type { return p.GetType() }, e.Items),
				},
				pattern: pattern,
			})

			for _, item := range e.Items {
				eqs = equatizePattern(eqs, item, loc)
			}
			break
		}
	default:
		panic(common.SystemError{Message: "invalid case"})
	}
	return eqs
}

func equatizeExpression(
	eqs []equation, expr typed.Expression, localDefs localDefsMap, stack []*typed.Definition, loc *ast.Location,
) []equation {
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
				loc:   loc,
				left:  e.Record.GetType(),
				right: &typed.TRecord{Location: e.Location, Fields: fields},
				expr:  expr,
			})
			eqs = equatizeExpression(eqs, e.Record, localDefs, stack, loc)
			break
		}
	case *typed.Apply:
		{
			e := expr.(*typed.Apply)
			eqs = append(eqs, equation{
				loc:  loc,
				left: e.Func.GetType(),
				right: &typed.TFunc{
					Location: e.Location,
					Params:   common.Map(func(p typed.Expression) typed.Type { return p.GetType() }, e.Args),
					Return:   e.Type,
				},
				expr: expr,
			})
			eqs = equatizeExpression(eqs, e.Func, localDefs, stack, loc)
			for _, a := range e.Args {
				eqs = equatizeExpression(eqs, a, localDefs, stack, loc)
			}
			break
		}
	case *typed.Const:
		{
			e := expr.(*typed.Const)
			eqs = append(eqs, equation{
				loc:   loc,
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
					loc:   loc,
					left:  e.Condition.GetType(),
					right: &typed.TExternal{Location: e.Location, Name: common.OakCoreBasicsBool},
					expr:  expr,
				},
				equation{
					loc:   loc,
					left:  e.Type,
					right: e.Positive.GetType(),
					expr:  expr,
				},
				equation{
					loc:   loc,
					left:  e.Type,
					right: e.Negative.GetType(),
					expr:  expr,
				})
			eqs = equatizeExpression(eqs, e.Condition, localDefs, stack, loc)
			eqs = equatizeExpression(eqs, e.Positive, localDefs, stack, loc)
			eqs = equatizeExpression(eqs, e.Negative, localDefs, stack, loc)
			break
		}
	case *typed.Let:
		{
			e := expr.(*typed.Let)
			eqs = append(eqs,
				equation{
					loc:   loc,
					left:  e.Type,
					right: e.Body.GetType(),
					expr:  expr,
				})
			eqs = equatizePattern(eqs, e.Pattern, loc)
			eqs = equatizeExpression(eqs, e.Value, localDefs, stack, loc)
			eqs = equatizeExpression(eqs, e.Body, localDefs, stack, loc)
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
						loc:   loc,
						left:  listItemType,
						right: item.GetType(),
						expr:  expr,
					})
				}
			}

			if listItemType == nil {
				listItemType = annotateType(nil, nil, e.Location, false)
			}
			eqs = append(eqs, equation{
				loc:  loc,
				left: e.Type,
				right: &typed.TExternal{
					Location: e.Location,
					Name:     common.OakCoreListList,
					Args:     []typed.Type{listItemType},
				},
				expr: expr,
			})

			for _, item := range e.Items {
				eqs = equatizeExpression(eqs, item, localDefs, stack, loc)
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
				loc:  loc,
				left: e.Type,
				right: &typed.TRecord{
					Location: e.Location,
					Fields:   fieldTypes,
				},
				expr: expr,
			})

			for _, f := range e.Fields {
				eqs = equatizeExpression(eqs, f.Value, localDefs, stack, loc)
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
						loc:   loc,
						left:  e.Condition.GetType(),
						right: cs.Pattern.GetType(),
						expr:  expr,
					}, equation{
						loc:   loc,
						left:  caseType,
						right: cs.Expression.GetType(),
						expr:  expr,
					})
			}

			for _, cs := range e.Cases {
				eqs = equatizePattern(eqs, cs.Pattern, loc)
				eqs = equatizeExpression(eqs, cs.Expression, localDefs, stack, loc)
			}
			break
		}
	case *typed.Tuple:
		{
			e := expr.(*typed.Tuple)
			eqs = append(eqs, equation{
				loc:  loc,
				left: e.Type,
				right: &typed.TTuple{
					Location: e.Location,
					Items:    common.Map(func(e typed.Expression) typed.Type { return e.GetType() }, e.Items),
				},
				expr: expr,
			})
			for _, item := range e.Items {
				eqs = equatizeExpression(eqs, item, localDefs, stack, loc)
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
				loc:  loc,
				left: e.Type,
				right: &typed.TRecord{
					Location: e.Location,
					Fields:   fieldTypes,
				},
				expr: expr,
			})

			for _, f := range e.Fields {
				eqs = equatizeExpression(eqs, f.Value, localDefs, stack, loc)
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
				loc:  loc,
				left: e.Type,
				right: &typed.TRecord{
					Location: e.Location,
					Fields:   fieldTypes,
				},
				expr: expr,
			})

			for _, f := range e.Fields {
				eqs = equatizeExpression(eqs, f.Value, localDefs, stack, loc)
			}
			eqs = equatizeDefinition(eqs, e.Definition, localDefsMap{}, stack, &e.Location)
			break
		}
	case *typed.Constructor:
		{
			e := expr.(*typed.Constructor)
			eqs = append(eqs, equation{
				loc:  loc,
				left: e.Type,
				right: &typed.TExternal{
					Location: e.Location,
					Name:     e.DataName,
				},
				expr: e,
			})
			for _, a := range e.Args {
				eqs = equatizeExpression(eqs, a, localDefs, stack, loc)
			}
			break
		}
	case *typed.NativeCall:
		{
			e := expr.(*typed.NativeCall)
			for _, a := range e.Args {
				eqs = equatizeExpression(eqs, a, localDefs, stack, loc)
			}
			break
		}
	case *typed.Local:
		{
			e := expr.(*typed.Local)
			if ld, ok := localDefs[e.Name]; ok {
				eqs = equatizeDefinition(eqs, ld, maps.Clone(localDefs), stack, &e.Location)
				eqs = append(eqs, equation{
					loc:   loc,
					left:  e.Type,
					right: ld.GetType(),
					expr:  e,
				})
			}
		}
	case *typed.Global:
		{
			e := expr.(*typed.Global)
			eqs = append(eqs, equation{
				loc:   loc,
				left:  e.Type,
				right: e.Definition.GetType(),
				expr:  e,
			})
			eqs = equatizeDefinition(eqs, e.Definition, localDefsMap{}, stack, &e.Location)
			break
		}
	default:
		panic(common.SystemError{Message: "invalid case"})
	}
	return eqs
}

func getConstType(cv ast.ConstValue, location ast.Location) typed.Type {
	switch cv.(type) {
	case ast.CChar:
		return &typed.TExternal{Location: location, Name: common.OakCoreCharChar}
	case ast.CInt:
		return newAnnotatedType(location, typed.ConstraintNumber)
	case ast.CFloat:
		return &typed.TExternal{Location: location, Name: common.OakCoreBasicsFloat}
	case ast.CString:
		return &typed.TExternal{Location: location, Name: common.OakCoreStringString}
	case ast.CUnit:
		return &typed.TExternal{Location: location, Name: common.OakCoreBasicsUnit}
	}
	panic(common.SystemError{Message: "invalid case"})
}

func unifyAll(eqs []equation, loc []ast.Location) map[uint64]typed.Type {
	var i int
	defer func() {
		err := recover()
		if err != nil {
			panic(err)
		}
	}()
	subst := map[uint64]typed.Type{}
	for _, eq := range eqs {
		var extra []ast.Location
		if eq.loc != nil {
			extra = append(extra, *eq.loc)
		}
		if eq.left != nil {
			extra = append(extra, eq.left.GetLocation())
		}
		if eq.expr != nil {
			extra = append(extra, eq.expr.GetLocation())
		}
		if eq.pattern != nil {
			extra = append(extra, eq.pattern.GetLocation())
		}
		if eq.def != nil {
			extra = append(extra, eq.def.Location)
		}

		unify(eq.left, eq.right, append(loc, extra...), subst)
		i++
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

func unify(x typed.Type, y typed.Type, loc []ast.Location, subst map[uint64]typed.Type) {
	if x.EqualsTo(y) {
		return
	}

	_, ubx := x.(*typed.TUnbound)
	_, uby := y.(*typed.TUnbound)

	if ubx {
		unifyUnbound(x.(*typed.TUnbound), y, loc, subst)
		return
	}
	if uby {
		unifyUnbound(y.(*typed.TUnbound), x, loc, subst)
		return
	}
	switch x.(type) {
	case *typed.TFunc:
		{
			if ey, ok := y.(*typed.TFunc); ok {
				ex := x.(*typed.TFunc)
				if len(ex.Params) < len(ey.Params) {
					ex, ey = ey, ex
				}
				ex = balanceFn(ex, len(ey.Params))
				for i, p := range ex.Params {
					unify(p, ey.Params[i], append(loc, p.GetLocation(), ey.Params[i].GetLocation()), subst)
				}
				unify(ex.Return, ey.Return, append(loc, ex.GetLocation(), ey.GetLocation()), subst)
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
					panic(common.Error{
						Extra:   []ast.Location{ex.Location, ey.Location},
						Message: "record fields number mismatch"})
				}
				for i, f := range ex.Fields {
					unify(f, ey.Fields[i], append(loc, f.GetLocation(), ey.Fields[i].GetLocation()), subst)
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
					panic(common.Error{
						Extra:   []ast.Location{ex.Location, ey.Location},
						Message: "tuple sizes mismatch"})
				}
				for i, p := range ex.Items {
					unify(p, ey.Items[i], append(loc, p.GetLocation(), ey.Items[i].GetLocation()), subst)
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
							unify(p, ey.Args[i], append(loc, p.GetLocation(), ey.Args[i].GetLocation()), subst)
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
	panic(common.Error{
		Extra:   append(loc, x.GetLocation(), y.GetLocation()),
		Message: fmt.Sprintf("%v cannot be matched with %v", x, y),
	})
}

func unifyUnbound(v *typed.TUnbound, typ typed.Type, loc []ast.Location, subst map[uint64]typed.Type) {
	if x, ok := subst[v.Index]; ok {
		unify(x, typ, loc, subst)
		return
	} else {
		if y, ok := typ.(*typed.TUnbound); ok {
			if uy, c := subst[y.Index]; c {
				unify(v, uy, loc, subst)
				return
			}
		}
		if OccursCheck(v, typ, subst) {
			panic(common.Error{
				Extra:   append(loc, v.Location, typ.GetLocation()),
				Message: fmt.Sprintf("ambigous type: %v vs %v", applyType(v, subst), applyType(typ, subst)),
			})
		}
	}
	subst[v.Index] = typ
}

func OccursCheck(v *typed.TUnbound, typ typed.Type, subst map[uint64]typed.Type) bool {
	if v.EqualsTo(typ) {
		return true
	}
	switch typ.(type) {
	case *typed.TFunc:
		{
			e := typ.(*typed.TFunc)
			if OccursCheck(v, e.Return, subst) {
				return true
			}
			for _, p := range e.Params {
				if OccursCheck(v, p, subst) {
					return true
				}
			}
			break
		}
	case *typed.TRecord:
		{
			e := typ.(*typed.TRecord)
			for _, f := range e.Fields {
				if OccursCheck(v, f, subst) {
					return true
				}
			}
			break
		}
	case *typed.TTuple:
		{
			e := typ.(*typed.TTuple)
			for _, i := range e.Items {
				if OccursCheck(v, i, subst) {
					return true
				}
			}
			break
		}
	case *typed.TExternal:
		{
			e := typ.(*typed.TExternal)
			for _, a := range e.Args {
				if OccursCheck(v, a, subst) {
					return true
				}
			}
			break
		}
	case *typed.TUnbound:
		{
			if c, ok := subst[typ.(*typed.TUnbound).Index]; ok {
				return OccursCheck(v, c, subst)
			}
			break
		}
	default:
		panic(common.SystemError{Message: "invalid case"})
	}
	return false
}

func applyDefinition(td *typed.Definition, subst map[uint64]typed.Type) *typed.Definition {
	td.Params = common.Map(func(p typed.Pattern) typed.Pattern {
		return applyPattern(p, subst)
	}, td.Params)
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
	case *typed.PDataOption:
		{
			e := pattern.(*typed.PDataOption)
			pattern = &typed.PDataOption{
				Location:   e.Location,
				Type:       applyType(e.Type, subst),
				Name:       e.Name,
				Definition: e.Definition,
				Args:       common.Map(apply, e.Args),
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
	case *typed.Apply:
		{
			e := expr.(*typed.Apply)
			expr = &typed.Apply{
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
				Location: e.Location,
				Type:     applyType(e.Type, subst),
				Pattern:  applyPattern(e.Pattern, subst),
				Value:    apply(e.Value),
				Body:     apply(e.Body),
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
	case *typed.UpdateGlobal:
		{
			e := expr.(*typed.UpdateGlobal)
			expr = &typed.UpdateGlobal{
				Location:       e.Location,
				Type:           applyType(e.Type, subst),
				ModuleName:     e.ModuleName,
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
	case *typed.Constructor:
		{
			e := expr.(*typed.Constructor)
			expr = &typed.Constructor{
				Location:   e.Location,
				Type:       applyType(e.Type, subst),
				DataName:   e.DataName,
				OptionName: e.OptionName,
				Args:       common.Map(apply, e.Args),
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
				ModuleName:     e.ModuleName,
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

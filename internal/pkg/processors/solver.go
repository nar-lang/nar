package processors

import (
	"fmt"
	"maps"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
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
type typeParamsMap map[ast.Identifier]typed.Type

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
) (errors []error) {
	if _, ok := typedModules[moduleName]; ok {
		return
	}

	m := modules[moduleName]

	for dep := range m.Dependencies {
		if dep != moduleName {
			if err := Solve(dep, modules, typedModules); err != nil {
				errors = append(errors, err...)
				return
			}
		}
	}

	o := typed.Module{
		Name:         m.Name,
		Location:     m.Location,
		Dependencies: m.Dependencies,
	}

	typedModules[o.Name] = &o

	for i := 0; i < len(m.Definitions); i++ {
		def := m.Definitions[i]
		if slices.ContainsFunc(o.Definitions, func(d *typed.Definition) bool { return d.Id == def.Id }) {
			continue
		}

		fp := fmt.Sprintf(".nar-bin/%s/%s.md", m.Name, def.Name)
		sb := strings.Builder{}

		unboundIndex = 0
		annotations = []struct {
			fmt.Stringer
			typed.Type
		}{}
		localTyped := map[ast.QualifiedIdentifier]*typed.Module{}
		var eqs []equation

		td, err := annotateDefinition(modules, localTyped, m.Name, def, nil)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		if dumpDebugOutput {
			_ = os.MkdirAll(filepath.Dir(fp), os.ModePerm)
			sb.WriteString(fmt.Sprintf("\n\nDefinition\n---\n`%s`", td))
			sb.WriteString("\n\nAnnotations\n---\n| Node | Type |\n|---|---|")
			for _, t := range annotations {
				sb.WriteString(fmt.Sprintf("\n| `%v` | `%v` |", t.Stringer, t.Type))
			}
			_ = os.WriteFile(fp, []byte(sb.String()), 0666)
		}

		eqs, err = equatizeDefinition(eqs, td, localDefsMap{}, nil, nil)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		if dumpDebugOutput {
			sb.WriteString("\n\nEquations\n---\n| No | Left | Right | Node |\n|---|---|---|---|")
			for i, eq := range eqs {
				sb.WriteString(eq.String(i))
			}
			_ = os.WriteFile(fp, []byte(sb.String()), 0666)
		}

		subst, err := unifyAll(eqs, []ast.Location{def.Location})

		if dumpDebugOutput {
			sb.WriteString("\n\nUnified\n---\n| Left | Right |\n|---|---|")
			for k, v := range subst {
				sb.WriteString(fmt.Sprintf("\n | `%v` | `%v` |", &typed.TUnbound{Index: k}, v))
			}
			_ = os.WriteFile(fp, []byte(sb.String()), 0666)
		}

		if err != nil {
			if _, ok := err.(common.Error); !ok {
				err = common.Error{
					Location: def.Location,
					Message:  fmt.Sprintf("failed while trying to solve type of %s.%s: %s", m.Name, def.Name, err.Error()),
				}
			}
			errors = append(errors, err)
			continue
		}

		td, err = applyDefinition(td, subst)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		if dumpDebugOutput {
			sb.WriteString("\n\nSolved\n---\n")
			sb.WriteString(fmt.Sprintf("\n `%v`", td.GetType()))
			sb.WriteString(fmt.Sprintf("\n `%v`", td))
			_ = os.WriteFile(fp, []byte(sb.String()), 0666)
		}

		o.Definitions = append(o.Definitions, td)
	}
	return
}

func annotateDefinition(
	modules map[ast.QualifiedIdentifier]*normalized.Module,
	typedModules map[ast.QualifiedIdentifier]*typed.Module,
	moduleName ast.QualifiedIdentifier,
	def *normalized.Definition,
	stack []*typed.Definition,
) (*typed.Definition, error) {
	o := &typed.Definition{
		Id:       def.Id,
		Name:     def.Name,
		Hidden:   def.Hidden,
		Location: def.Location,
	}

	localSymbols := symbolsMap{}
	localTypeParams := typeParamsMap{}

	var err error
	o.Params, err = common.MapError(
		func(p normalized.Pattern) (typed.Pattern, error) {
			return annotatePattern(localSymbols, localTypeParams, modules, typedModules, moduleName, p, true, stack)
		},
		def.Params)
	if err != nil {
		return nil, err
	}

	if def.Type != nil {
		o.DefinedType, err = annotateType("", localTypeParams, def.Type, def.Location, true, placeholderMap{})
		if err != nil {
			return nil, err
		}
	}

	for _, std := range stack {
		if std.Id == def.Id {
			return std, nil
		}
	}

	stack = append(stack, o)
	o.Expression, err = annotateExpression(
		localSymbols, localTypeParams, modules, typedModules, moduleName, def.Expression, stack)
	if err != nil {
		return nil, err
	}
	stack = stack[:len(stack)-1]
	return o, nil
}

func annotatePattern(symbols symbolsMap,
	typeParams typeParamsMap,
	modules map[ast.QualifiedIdentifier]*normalized.Module,
	typedModules map[ast.QualifiedIdentifier]*typed.Module,
	moduleName ast.QualifiedIdentifier,
	pattern normalized.Pattern,
	typeMapSource bool,
	stack []*typed.Definition,
) (typed.Pattern, error) {
	if pattern == nil {
		return nil, nil
	}
	annotate := func(p normalized.Pattern) (typed.Pattern, error) {
		return annotatePattern(symbols, typeParams, modules, typedModules, moduleName, p, typeMapSource, stack)
	}
	var p typed.Pattern
	switch pattern.(type) {
	case normalized.PAlias:
		{
			e := pattern.(normalized.PAlias)
			nested, err := annotate(e.Nested)
			if err != nil {
				return nil, err
			}
			type_, err := annotateType("", typeParams, e.Type, e.Location, typeMapSource, placeholderMap{})
			if err != nil {
				return nil, err
			}
			p = &typed.PAlias{
				Location: e.Location,
				Type:     type_,
				Alias:    e.Alias,
				Nested:   nested,
			}
			symbols[e.Alias] = p.GetType()
			break
		}
	case normalized.PAny:
		{
			e := pattern.(normalized.PAny)
			type_, err := annotateType("", typeParams, e.Type, e.Location, typeMapSource, placeholderMap{})
			if err != nil {
				return nil, err
			}
			p = &typed.PAny{
				Location: e.Location,
				Type:     type_,
			}
			break
		}
	case normalized.PCons:
		{
			e := pattern.(normalized.PCons)
			head, err := annotate(e.Head)
			if err != nil {
				return nil, err
			}
			tail, err := annotate(e.Tail)
			if err != nil {
				return nil, err
			}
			type_, err := annotateType("", typeParams, e.Type, e.Location, typeMapSource, placeholderMap{})
			if err != nil {
				return nil, err
			}
			p = &typed.PCons{
				Location: e.Location,
				Type:     type_,
				Head:     head,
				Tail:     tail,
			}
			break
		}
	case normalized.PConst:
		{
			e := pattern.(normalized.PConst)
			type_, err := annotateType("", typeParams, e.Type, e.Location, typeMapSource, placeholderMap{})
			if err != nil {
				return nil, err
			}
			p = &typed.PConst{
				Location: e.Location,
				Type:     type_,
				Value:    e.Value,
			}
			break
		}
	case normalized.PDataOption:
		{
			e := pattern.(normalized.PDataOption)
			def, err := getAnnotatedGlobal(e.ModuleName, e.DefinitionName, modules, typedModules, stack, e.Location)
			if err != nil {
				return nil, err
			}
			if ctor, ok := def.Expression.(*typed.Constructor); !ok {
				return nil, common.NewCompilerError("data option definition is not a constructor")
			} else {
				args, err := common.MapError(annotate, e.Values)
				if err != nil {
					return nil, err
				}
				type_, err := annotateType("", typeParams, nil, e.Location, typeMapSource, placeholderMap{})
				if err != nil {
					return nil, err
				}
				p = &typed.PDataOption{
					Location:   e.Location,
					Type:       type_,
					DataName:   ctor.DataName,
					OptionName: ctor.OptionName,
					Args:       args,
					Definition: def,
				}
			}
			break
		}
	case normalized.PList:
		{
			e := pattern.(normalized.PList)
			items, err := common.MapError(annotate, e.Items)
			if err != nil {
				return nil, err
			}
			type_, err := annotateType("", typeParams, e.Type, e.Location, typeMapSource, placeholderMap{})
			if err != nil {
				return nil, err
			}
			p = &typed.PList{
				Location: e.Location,
				Type:     type_,
				Items:    items,
			}
			break
		}
	case normalized.PNamed:
		{
			e := pattern.(normalized.PNamed)
			type_, err := annotateType("", typeParams, nil, e.Location, typeMapSource, placeholderMap{})
			if err != nil {
				return nil, err
			}
			p = &typed.PNamed{
				Location: e.Location,
				Type:     type_,
				Name:     e.Name,
			}
			symbols[e.Name] = p.GetType()
			break
		}
	case normalized.PRecord:
		{
			e := pattern.(normalized.PRecord)
			type_, err := annotateType("", typeParams, e.Type, e.Location, typeMapSource, placeholderMap{})
			if err != nil {
				return nil, err
			}
			fields, err := common.MapError(func(f normalized.PRecordField) (typed.PRecordField, error) {
				type_, err := annotateType("", typeParams, nil, e.Location, typeMapSource, placeholderMap{})
				if err != nil {
					return typed.PRecordField{}, err
				}
				return typed.PRecordField{
					Location: f.Location,
					Name:     f.Name,
					Type:     type_,
				}, nil
			}, e.Fields)
			if err != nil {
				return nil, err
			}
			p = &typed.PRecord{
				Location: e.Location,
				Type:     type_,
				Fields:   fields,
			}
			break
		}
	case normalized.PTuple:
		{
			e := pattern.(normalized.PTuple)
			items, err := common.MapError(annotate, e.Items)
			if err != nil {
				return nil, err
			}
			type_, err := annotateType("", typeParams, e.Type, e.Location, typeMapSource, placeholderMap{})
			if err != nil {
				return nil, err
			}
			p = &typed.PTuple{
				Location: e.Location,
				Type:     type_,
				Items:    items,
			}
			break
		}
	default:
		return nil, common.NewCompilerError("impossible case")
	}

	annotations = append(annotations, struct {
		fmt.Stringer
		typed.Type
	}{p, p.GetType()})
	return p, nil
}

func annotateExpression(
	symbols symbolsMap,
	typeParams typeParamsMap,
	modules map[ast.QualifiedIdentifier]*normalized.Module,
	typedModules map[ast.QualifiedIdentifier]*typed.Module,
	moduleName ast.QualifiedIdentifier,
	expr normalized.Expression,
	stack []*typed.Definition,
) (typed.Expression, error) {
	if expr == nil {
		return nil, nil
	}

	annotate := func(e normalized.Expression) (typed.Expression, error) {
		return annotateExpression(symbols, typeParams, modules, typedModules, moduleName, e, stack)
	}
	var o typed.Expression
	switch expr.(type) {
	case normalized.Access:
		{
			e := expr.(normalized.Access)
			record, err := annotate(e.Record)
			if err != nil {
				return nil, err
			}
			type_, err := annotateType("", typeParams, nil, e.Location, false, placeholderMap{})
			if err != nil {
				return nil, err
			}
			o = &typed.Access{
				Location:  e.Location,
				Type:      type_,
				Record:    record,
				FieldName: e.FieldName,
			}
			break
		}
	case normalized.Apply:
		{
			e := expr.(normalized.Apply)
			fn, err := annotate(e.Func)
			if err != nil {
				return nil, err
			}
			args, err := common.MapError(annotate, e.Args)
			if err != nil {
				return nil, err
			}
			type_, err := annotateType("", typeParams, nil, e.Location, false, placeholderMap{})
			if err != nil {
				return nil, err
			}
			o = &typed.Apply{
				Location: e.Location,
				Type:     type_,
				Func:     fn,
				Args:     args,
			}
			break
		}
	case normalized.Const:
		{
			e := expr.(normalized.Const)
			type_, err := annotateType("", typeParams, nil, e.Location, false, placeholderMap{})
			if err != nil {
				return nil, err
			}
			o = &typed.Const{
				Location: e.Location,
				Type:     type_,
				Value:    e.Value,
			}
			break
		}
	case normalized.LetMatch:
		{
			e := expr.(normalized.LetMatch)

			localSymbols := maps.Clone(symbols)
			localTypeParams := maps.Clone(typeParams)

			pattern, err := annotatePattern(
				localSymbols, localTypeParams, modules, typedModules, moduleName, e.Pattern, true, stack)
			if err != nil {
				return nil, err
			}
			value, err := annotateExpression(
				localSymbols, localTypeParams, modules, typedModules, moduleName, e.Value, stack)
			if err != nil {
				return nil, err
			}
			body, err := annotateExpression(
				localSymbols, localTypeParams, modules, typedModules, moduleName, e.Nested, stack)
			if err != nil {
				return nil, err
			}
			type_, err := annotateType("", localTypeParams, nil, e.Location, true, placeholderMap{})
			if err != nil {
				return nil, err
			}
			o = &typed.Let{
				Location: e.Location,
				Type:     type_,
				Pattern:  pattern,
				Value:    value,
				Body:     body,
			}
			break
		}
	case normalized.List:
		{
			e := expr.(normalized.List)
			items, err := common.MapError(annotate, e.Items)
			if err != nil {
				return nil, err
			}
			type_, err := annotateType("", typeParams, nil, e.Location, false, placeholderMap{})
			if err != nil {
				return nil, err
			}
			o = &typed.List{
				Location: e.Location,
				Type:     type_,
				Items:    items,
			}
			break
		}
	case normalized.Record:
		{
			e := expr.(normalized.Record)
			fields, err := common.MapError(func(f normalized.RecordField) (typed.RecordField, error) {
				value, err := annotate(f.Value)
				if err != nil {
					return typed.RecordField{}, err
				}
				type_, err := annotateType("", typeParams, nil, f.Location, false, placeholderMap{})
				if err != nil {
					return typed.RecordField{}, err
				}
				return typed.RecordField{
					Location: e.Location,
					Type:     type_,
					Name:     f.Name,
					Value:    value,
				}, nil
			}, e.Fields)
			if err != nil {
				return nil, err
			}
			type_, err := annotateType("", typeParams, nil, e.Location, false, placeholderMap{})
			if err != nil {
				return nil, err
			}
			o = &typed.Record{
				Location: e.Location,
				Type:     type_,
				Fields:   fields,
			}
			break
		}
	case normalized.Select:
		{
			e := expr.(normalized.Select)
			condition, err := annotate(e.Condition)
			if err != nil {
				return nil, err
			}
			cases, err := common.MapError(func(c normalized.SelectCase) (typed.SelectCase, error) {
				localSymbols := maps.Clone(symbols)
				localTypeParams := maps.Clone(typeParams)
				pattern, err := annotatePattern(
					localSymbols, localTypeParams, modules, typedModules, moduleName, c.Pattern, false, stack)
				if err != nil {
					return typed.SelectCase{}, err
				}
				expr, err := annotateExpression(
					localSymbols, localTypeParams, modules, typedModules, moduleName, c.Expression, stack)
				if err != nil {
					return typed.SelectCase{}, err
				}
				type_, err := annotateType("", localTypeParams, nil, c.Location, false, placeholderMap{})
				if err != nil {
					return typed.SelectCase{}, err
				}
				return typed.SelectCase{
					Location:   c.Location,
					Pattern:    pattern,
					Expression: expr,
					Type:       type_,
				}, nil
			}, e.Cases)
			if err != nil {
				return nil, err
			}
			type_, err := annotateType("", typeParams, nil, e.Location, false, placeholderMap{})
			if err != nil {
				return nil, err
			}
			o = &typed.Select{
				Location:  e.Location,
				Type:      type_,
				Condition: condition,
				Cases:     cases,
			}
			break
		}
	case normalized.Tuple:
		{
			e := expr.(normalized.Tuple)
			items, err := common.MapError(annotate, e.Items)
			if err != nil {
				return nil, err
			}
			type_, err := annotateType("", typeParams, nil, e.Location, false, placeholderMap{})
			if err != nil {
				return nil, err
			}
			o = &typed.Tuple{
				Location: e.Location,
				Type:     type_,
				Items:    items,
			}
			break
		}
	case normalized.UpdateLocal:
		{
			e := expr.(normalized.UpdateLocal)
			if t, ok := symbols[e.RecordName]; ok {
				fields, err := common.MapError(func(f normalized.RecordField) (typed.RecordField, error) {
					value, err := annotate(f.Value)
					if err != nil {
						return typed.RecordField{}, err
					}
					type_, err := annotateType("", typeParams, nil, f.Location, false, placeholderMap{})
					if err != nil {
						return typed.RecordField{}, err
					}
					return typed.RecordField{
						Location: e.Location,
						Type:     type_,
						Name:     f.Name,
						Value:    value,
					}, nil
				}, e.Fields)
				if err != nil {
					return nil, err
				}
				o = &typed.UpdateLocal{
					Location:   e.Location,
					Type:       t,
					RecordName: e.RecordName,
					Fields:     fields,
				}
			} else {
				return nil, common.Error{
					Location: e.Location,
					Message:  fmt.Sprintf("local variable `%s` not found", e.RecordName),
				}
			}
			break
		}
	case normalized.UpdateGlobal:
		{
			e := expr.(normalized.UpdateGlobal)

			def, err := getAnnotatedGlobal(e.ModuleName, e.DefinitionName, modules, typedModules, stack, e.Location)
			if err != nil {
				return nil, err
			}
			fields, err := common.MapError(func(f normalized.RecordField) (typed.RecordField, error) {
				value, err := annotate(f.Value)
				if err != nil {
					return typed.RecordField{}, err
				}
				type_, err := annotateType("", typeParams, nil, f.Location, false, placeholderMap{})
				if err != nil {
					return typed.RecordField{}, err
				}
				return typed.RecordField{
					Location: e.Location,
					Type:     type_,
					Name:     f.Name,
					Value:    value,
				}, nil
			}, e.Fields)
			if err != nil {
				return nil, err
			}

			o = &typed.UpdateGlobal{
				Location:       e.Location,
				Type:           def.GetType(),
				ModuleName:     e.ModuleName,
				DefinitionName: e.DefinitionName,
				Definition:     def,
				Fields:         fields,
			}
			break
		}
	case normalized.Constructor:
		{
			e := expr.(normalized.Constructor)
			def, err := getAnnotatedGlobal(e.ModuleName, e.OptionName, modules, typedModules, stack, e.Location)
			if err != nil {
				return nil, err
			}
			t := def.DefinedType
			if len(def.Params) > 0 && t != nil {
				t = t.(*typed.TFunc).Return
			}
			args, err := common.MapError(annotate, e.Args)
			if err != nil {
				return nil, err
			}
			type_, err := annotateType("", typeParams, nil, e.Location, false, placeholderMap{})
			if err != nil {
				return nil, err
			}
			var dt *typed.TData
			if t != nil {
				dt = t.(*typed.TData)
			}
			o = &typed.Constructor{
				Location:   e.Location,
				Type:       type_,
				DataName:   common.MakeFullIdentifier(e.ModuleName, e.DataName),
				OptionName: e.OptionName,
				DataType:   dt,
				Args:       args,
			}
			break
		}
	case normalized.NativeCall:
		{
			e := expr.(normalized.NativeCall)
			args, err := common.MapError(annotate, e.Args)
			if err != nil {
				return nil, err
			}
			type_, err := annotateType("", typeParams, nil, e.Location, false, placeholderMap{})
			if err != nil {
				return nil, err
			}
			o = &typed.NativeCall{
				Location: e.Location,
				Type:     type_,
				Name:     e.Name,
				Args:     args,
			}
			break
		}
	case normalized.Local:
		{
			e := expr.(normalized.Local)
			localType, ok := symbols[e.Name]
			if !ok {
				return nil, common.Error{
					Location: e.Location, Message: fmt.Sprintf("local variable `%s` not found", e.Name),
				}
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
			def, err := getAnnotatedGlobal(e.ModuleName, e.DefinitionName, modules, typedModules, stack, e.Location)
			if err != nil {
				return nil, err
			}

			dt := def.GetType()
			if dt == nil {
				dt, err = annotateType("", typeParams, nil, e.Location, false, placeholderMap{})
				if err != nil {
					return nil, err
				}
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
		return nil, common.NewCompilerError("impossible case")
	}

	annotations = append(annotations, struct {
		fmt.Stringer
		typed.Type
	}{o, o.GetType()})
	return o, nil
}

func getAnnotatedGlobal(
	moduleName ast.QualifiedIdentifier,
	definitionName ast.Identifier,
	modules map[ast.QualifiedIdentifier]*normalized.Module,
	typedModules map[ast.QualifiedIdentifier]*typed.Module,
	stack []*typed.Definition,
	loc ast.Location,
) (*typed.Definition, error) {
	mod, ok := modules[moduleName]
	if !ok {
		return nil, common.Error{
			Location: loc,
			Message:  fmt.Sprintf("module `%s` not found", moduleName),
		}
	}
	nDef, ok := common.Find(
		func(definition *normalized.Definition) bool {
			return definition.Name == definitionName
		},
		mod.Definitions)
	if !ok {
		return nil, common.Error{
			Location: loc,
			Message:  fmt.Sprintf("definition `%s` not found", definitionName),
		}
	}

	def, ok := common.Find(func(definition *typed.Definition) bool {
		return definition.Id == nDef.Id
	}, stack)

	if !ok {
		var err error
		def, err = annotateDefinition(modules, typedModules, moduleName, nDef, stack)
		if err != nil {
			return def, err
		}
	}

	return def, nil
}

func newAnnotatedType(loc ast.Location, constraint common.Constraint) typed.Type {
	unboundIndex++
	return &typed.TUnbound{
		Location:   loc,
		Index:      unboundIndex,
		Constraint: constraint,
	}
}

type placeholderMap map[ast.FullIdentifier]typed.Type

func annotateType(
	name ast.Identifier, typeParams typeParamsMap, t normalized.Type, location ast.Location, typeMapSource bool,
	placeholders placeholderMap,
) (typed.Type, error) {
	annotate := func(l ast.Location) func(x normalized.Type) (typed.Type, error) {
		return func(x normalized.Type) (typed.Type, error) {
			return annotateType("", typeParams, x, location, typeMapSource, placeholders)
		}
	}

	var r typed.Type
	if t == nil {
		constraint := common.ConstraintNone
		if strings.HasPrefix(string(name), string(common.ConstraintNumber)) {
			constraint = common.ConstraintNumber
		}
		r = newAnnotatedType(location, constraint)

	} else {
		switch t.(type) {
		case *normalized.TFunc:
			{
				e := t.(*normalized.TFunc)
				params, err := common.MapError(annotate(e.Location), e.Params)
				if err != nil {
					return nil, err
				}
				ret, err := annotateType("", typeParams, e.Return, e.Location, typeMapSource, placeholders)
				if err != nil {
					return nil, err
				}
				r = &typed.TFunc{
					Location: e.Location,
					Params:   params,
					Return:   ret,
				}
				break
			}
		case *normalized.TRecord:
			{
				e := t.(*normalized.TRecord)
				fields := map[ast.Identifier]typed.Type{}
				for n, v := range e.Fields {
					var err error
					fields[n], err = annotateType("", typeParams, v, e.Location, typeMapSource, placeholders)
					if err != nil {
						return nil, err
					}
				}
				r = &typed.TRecord{
					Location: e.Location,
					Fields:   fields,
				}
				break
			}
		case *normalized.TTuple:
			{
				e := t.(*normalized.TTuple)
				items, err := common.MapError(annotate(e.Location), e.Items)
				if err != nil {
					return nil, err
				}
				r = &typed.TTuple{
					Location: e.Location,
					Items:    items,
				}
				break
			}
		case *normalized.TUnit:
			{
				e := t.(*normalized.TUnit)
				r = &typed.TNative{Location: e.Location, Name: common.NarCoreBasicsUnit}
				break
			}
		case *normalized.TData:
			{
				e := t.(*normalized.TData)
				d := &typed.TData{
					Location: e.Location,
					Name:     e.Name,
				}
				placeholders[d.Name] = d
				var err error
				d.Args, err = common.MapError(annotate(e.Location), e.Args)
				if err != nil {
					return nil, err
				}
				d.Options, err = common.MapError(
					func(x normalized.DataOption) (typed.DataOption, error) {
						values, err := common.MapError(annotate(e.Location), x.Values)
						if err != nil {
							return typed.DataOption{}, err
						}
						return typed.DataOption{
							Name:   common.MakeDataOptionIdentifier(e.Name, x.Name),
							Values: values,
						}, nil
					},
					e.Options)
				if err != nil {
					return nil, err
				}
				r = d
				break
			}
		case *normalized.TNative:
			{
				e := t.(*normalized.TNative)
				args, err := common.MapError(annotate(e.Location), e.Args)
				if err != nil {
					return nil, err
				}
				r = &typed.TNative{
					Location: e.Location,
					Name:     e.Name,
					Args:     args,
				}
				break
			}
		case *normalized.TTypeParameter:
			{
				e := t.(*normalized.TTypeParameter)

				if id, ok := typeParams[e.Name]; ok {
					r = id
				} else {
					if typeMapSource {
						var err error
						r, err = annotateType(e.Name, typeParams, nil, e.Location, true, placeholders)
						if err != nil {
							return nil, err
						}
						annotations = append(annotations, struct {
							fmt.Stringer
							typed.Type
						}{e, r})
						typeParams[e.Name] = r
					} else {
						return nil, common.Error{
							Location: e.Location, Message: "unknown type parameter",
						}
					}
				}
				break
			}
		case *normalized.TPlaceholder:
			{
				e := t.(*normalized.TPlaceholder)
				if p, ok := placeholders[e.Name]; ok {
					r = p
				} else {

					placeholders[e.Name] = r
				}
			}
		default:
			return nil, common.NewCompilerError("impossible case")

		}
	}
	return r, nil
}

func equatizeDefinition(
	eqs []equation, td *typed.Definition, localDefs localDefsMap, stack []*typed.Definition, loc *ast.Location,
) ([]equation, error) {
	for _, std := range stack {
		if std.Id == td.Id {
			return eqs, nil
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

	var err error
	for _, p := range td.Params {
		eqs, err = equatizePattern(eqs, p, stack, loc)
		if err != nil {
			return nil, err
		}
	}

	if td.Expression != nil {
		eqs, err = equatizeExpression(eqs, td.Expression, localDefs, stack, loc)
		if err != nil {
			return nil, err
		}
	}

	stack = stack[:len(stack)-1]
	return eqs, nil
}

func equatizePattern(
	eqs []equation, pattern typed.Pattern, stack []*typed.Definition, loc *ast.Location,
) ([]equation, error) {
	var err error
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
			eqs, err = equatizePattern(eqs, e.Nested, stack, loc)
			if err != nil {
				return nil, err
			}
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
					right: &typed.TNative{
						Location: e.Location,
						Name:     common.NarCoreListList,
						Args:     []typed.Type{e.Head.GetType()},
					},
					pattern: pattern,
				})
			eqs, err = equatizePattern(eqs, e.Head, stack, loc)
			if err != nil {
				return nil, err
			}
			eqs, err = equatizePattern(eqs, e.Tail, stack, loc)
			if err != nil {
				return nil, err
			}
			break
		}
	case *typed.PConst:
		{
			e := pattern.(*typed.PConst)
			const_, err := getConstType(e.Value, e.Location)
			if err != nil {
				return nil, err
			}
			eqs = append(eqs, equation{
				loc:     loc,
				left:    e.Type,
				right:   const_,
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
					eqs, err = equatizePattern(eqs, arg, stack, loc)
					if err != nil {
						return nil, err
					}
				}
			}
			eqs, err = equatizeDefinition(eqs, e.Definition, localDefsMap{}, stack, &e.Location)
			if err != nil {
				return nil, err
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
				itemType, err = annotateType("", nil, nil, e.Location, false, placeholderMap{})
				if err != nil {
					return nil, err
				}
			}
			eqs = append(eqs, equation{
				loc:  loc,
				left: e.Type,
				right: &typed.TNative{
					Location: e.Location,
					Name:     common.NarCoreListList,
					Args:     []typed.Type{itemType},
				},
				pattern: pattern,
			})
			for _, item := range e.Items {
				eqs, err = equatizePattern(eqs, item, stack, loc)
				if err != nil {
					return nil, err
				}
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
					Location:          e.Location,
					Fields:            fields,
					MayHaveMoreFields: true,
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
				eqs, err = equatizePattern(eqs, item, stack, loc)
				if err != nil {
					return nil, err
				}
			}
			break
		}
	default:
		return nil, common.NewCompilerError("impossible case")
	}
	return eqs, nil
}

func equatizeExpression(
	eqs []equation, expr typed.Expression, localDefs localDefsMap, stack []*typed.Definition, loc *ast.Location,
) ([]equation, error) {
	if expr == nil {
		return eqs, nil
	}
	var err error
	switch expr.(type) {
	case *typed.Access:
		{
			e := expr.(*typed.Access)

			fields := map[ast.Identifier]typed.Type{}
			fields[e.FieldName] = e.Type
			eqs = append(eqs, equation{
				loc:   loc,
				left:  &typed.TRecord{Location: e.Location, Fields: fields, MayHaveMoreFields: true},
				right: e.Record.GetType(),
				expr:  expr,
			})
			eqs, err = equatizeExpression(eqs, e.Record, localDefs, stack, loc)
			if err != nil {
				return nil, err
			}
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
			eqs, err = equatizeExpression(eqs, e.Func, localDefs, stack, loc)
			if err != nil {
				return nil, err
			}
			for _, a := range e.Args {
				eqs, err = equatizeExpression(eqs, a, localDefs, stack, loc)
				if err != nil {
					return nil, err
				}
			}
			break
		}
	case *typed.Const:
		{
			e := expr.(*typed.Const)
			const_, err := getConstType(e.Value, e.Location)
			if err != nil {
				return nil, err
			}
			eqs = append(eqs, equation{
				loc:   loc,
				left:  e.Type,
				right: const_,
				expr:  e,
			})
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
			eqs, err = equatizePattern(eqs, e.Pattern, stack, loc)
			if err != nil {
				return nil, err
			}
			eqs, err = equatizeExpression(eqs, e.Value, localDefs, stack, loc)
			if err != nil {
				return nil, err
			}
			eqs, err = equatizeExpression(eqs, e.Body, localDefs, stack, loc)
			if err != nil {
				return nil, err
			}
			eqs = append(eqs,
				equation{
					loc:   loc,
					left:  e.Pattern.GetType(),
					right: e.Value.GetType(),
					expr:  expr,
				})
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
				listItemType, err = annotateType("", nil, nil, e.Location, false, placeholderMap{})
				if err != nil {
					return nil, err
				}
			}
			eqs = append(eqs, equation{
				loc:  loc,
				left: e.Type,
				right: &typed.TNative{
					Location: e.Location,
					Name:     common.NarCoreListList,
					Args:     []typed.Type{listItemType},
				},
				expr: expr,
			})

			for _, item := range e.Items {
				eqs, err = equatizeExpression(eqs, item, localDefs, stack, loc)
				if err != nil {
					return nil, err
				}
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
				eqs = append(eqs, equation{
					loc:   &f.Location,
					left:  f.Type,
					right: f.Value.GetType(),
					expr:  expr,
				})
			}

			for _, f := range e.Fields {
				eqs, err = equatizeExpression(eqs, f.Value, localDefs, stack, loc)
				if err != nil {
					return nil, err
				}
			}
			break
		}
	case *typed.Select:
		{
			e := expr.(*typed.Select)

			eqs, err = equatizeExpression(eqs, e.Condition, localDefs, stack, loc)
			if err != nil {
				return nil, err
			}
			for _, cs := range e.Cases {
				eqs = append(eqs,
					equation{
						loc:   loc,
						left:  e.Condition.GetType(),
						right: cs.Pattern.GetType(),
						expr:  expr,
					}, equation{
						loc:   loc,
						left:  e.Type,
						right: cs.Expression.GetType(),
						expr:  expr,
					})
			}

			for _, cs := range e.Cases {
				eqs, err = equatizePattern(eqs, cs.Pattern, stack, loc)
				if err != nil {
					return nil, err
				}
				eqs, err = equatizeExpression(eqs, cs.Expression, localDefs, stack, loc)
				if err != nil {
					return nil, err
				}
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
				eqs, err = equatizeExpression(eqs, item, localDefs, stack, loc)
				if err != nil {
					return nil, err
				}
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
				eqs = append(eqs, equation{
					loc:   &f.Location,
					left:  f.Type,
					right: f.Value.GetType(),
					expr:  expr,
				})
			}

			for _, f := range e.Fields {
				eqs, err = equatizeExpression(eqs, f.Value, localDefs, stack, loc)
				if err != nil {
					return nil, err
				}
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
				eqs = append(eqs, equation{
					loc:   &f.Location,
					left:  f.Type,
					right: f.Value.GetType(),
					expr:  expr,
				})
			}

			for _, f := range e.Fields {
				eqs, err = equatizeExpression(eqs, f.Value, localDefs, stack, loc)
				if err != nil {
					return nil, err
				}
			}
			eqs, err = equatizeDefinition(eqs, e.Definition, localDefsMap{}, stack, &e.Location)
			if err != nil {
				return nil, err
			}
			break
		}
	case *typed.Constructor:
		{
			e := expr.(*typed.Constructor)
			r := typed.TData{Location: e.Location, Name: e.DataName}
			if e.DataType != nil {
				r.Options = e.DataType.Options
				r.Args = e.DataType.Args
			}
			eqs = append(eqs, equation{
				loc:   loc,
				left:  e.Type,
				right: &r,
				expr:  e,
			})
			for _, a := range e.Args {
				eqs, err = equatizeExpression(eqs, a, localDefs, stack, loc)
				if err != nil {
					return nil, err
				}
			}
			break
		}
	case *typed.NativeCall:
		{
			e := expr.(*typed.NativeCall)
			for _, a := range e.Args {
				eqs, err = equatizeExpression(eqs, a, localDefs, stack, loc)
				if err != nil {
					return nil, err
				}
			}
			break
		}
	case *typed.Local:
		{
			e := expr.(*typed.Local)
			if ld, ok := localDefs[e.Name]; ok {
				eqs, err = equatizeDefinition(eqs, ld, maps.Clone(localDefs), stack, &e.Location)
				if err != nil {
					return nil, err
				}
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
			eqs, err = equatizeDefinition(eqs, e.Definition, localDefsMap{}, stack, &e.Location)
			if err != nil {
				return nil, err
			}
			break
		}
	default:
		return nil, common.NewCompilerError("impossible case")
	}
	return eqs, nil
}

func getConstType(cv ast.ConstValue, location ast.Location) (typed.Type, error) {
	switch cv.(type) {
	case ast.CChar:
		return &typed.TNative{Location: location, Name: common.NarCoreCharChar}, nil
	case ast.CInt:
		return newAnnotatedType(location, common.ConstraintNumber), nil
	case ast.CFloat:
		return &typed.TNative{Location: location, Name: common.NarCoreMathFloat}, nil
	case ast.CString:
		return &typed.TNative{Location: location, Name: common.NarCoreStringString}, nil
	case ast.CUnit:
		return &typed.TNative{Location: location, Name: common.NarCoreBasicsUnit}, nil
	}
	return nil, common.NewCompilerError("impossible case")
}

func unifyAll(eqs []equation, loc []ast.Location) (map[uint64]typed.Type, error) {
	var i int
	subst := map[uint64]typed.Type{}
	for _, eq := range eqs {
		if eq.left == nil || eq.right == nil {
			continue
		}
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

		err := unify(eq.left, eq.right, append(extra, loc...), subst)

		if err != nil {
			ce := err.(common.Error)
			if dumpDebugOutput {
				ce.Message += fmt.Sprintf(" (in equation %d)", i)
			}
			return subst, ce
		}
		i++
	}
	return subst, nil
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

func typesEqual(x typed.Type, y typed.Type, req map[ast.FullIdentifier]struct{}) bool {
	switch x.(type) {
	case *typed.TData:
		tx, okx := x.(*typed.TData)
		ty, oky := y.(*typed.TData)
		if okx && oky {
			if tx.Name != ty.Name {
				return false
			}
			if req != nil {
				if _, ok := req[tx.Name]; ok {
					return true
				}
			}
			req = map[ast.FullIdentifier]struct{}{}
			req[tx.Name] = struct{}{}
			if len(tx.Args) != len(ty.Args) {
				return false
			}
			for i, a := range tx.Args {
				if !typesEqual(a, ty.Args[i], req) {
					return false
				}
			}
			return true
		}
		break
	case *typed.TNative:
		tx, okx := x.(*typed.TNative)
		ty, oky := y.(*typed.TNative)
		if okx && oky {
			if tx.Name != ty.Name {
				return false
			}
			if len(tx.Args) != len(ty.Args) {
				return false
			}
			for i, a := range tx.Args {
				if !typesEqual(a, ty.Args[i], req) {
					return false
				}
			}
			return true
		}
		break
	case *typed.TFunc:
		tx, okx := x.(*typed.TFunc)
		ty, oky := y.(*typed.TFunc)
		if okx && oky {
			if len(tx.Params) != len(ty.Params) {
				return false
			}
			for i, p := range tx.Params {
				if !typesEqual(p, ty.Params[i], req) {
					return false
				}
			}
			return typesEqual(tx.Return, ty.Return, req)
		}
		break
	case *typed.TRecord:
		tx, okx := x.(*typed.TRecord)
		ty, oky := y.(*typed.TRecord)
		if okx && oky {
			if len(tx.Fields) != len(ty.Fields) {
				return false
			}
			for n, fx := range tx.Fields {
				if fy, ok := ty.Fields[n]; !ok {
					return false
				} else if !typesEqual(fx, fy, req) {
					return false
				}
			}
			return true
		}
		break
	case *typed.TTuple:
		tx, okx := x.(*typed.TTuple)
		ty, oky := y.(*typed.TTuple)
		if okx && oky {
			if len(tx.Items) != len(ty.Items) {
				return false
			}
			for i, p := range tx.Items {
				if !typesEqual(p, ty.Items[i], req) {
					return false
				}
			}
			return true
		}
		break
	case *typed.TUnbound:
		tx, okx := x.(*typed.TUnbound)
		ty, oky := y.(*typed.TUnbound)
		if okx && oky {
			return tx.Index == ty.Index && tx.Constraint == ty.Constraint
		}
	}
	return false
}

func unify(x typed.Type, y typed.Type, loc []ast.Location, subst map[uint64]typed.Type) error {
	if typesEqual(x, y, nil) {
		return nil
	}

	_, ubx := x.(*typed.TUnbound)
	_, uby := y.(*typed.TUnbound)

	if ubx {
		return unifyUnbound(x.(*typed.TUnbound), y, loc, subst)
	}
	if uby {
		return unifyUnbound(y.(*typed.TUnbound), x, loc, subst)
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
					err := unify(p, ey.Params[i], append(loc, p.GetLocation(), ey.Params[i].GetLocation()), subst)
					if err != nil {
						return err
					}
				}
				return unify(ex.Return, ey.Return, append(loc, ex.GetLocation(), ey.GetLocation()), subst)
			}
			break
		}
	case *typed.TRecord:
		{
			if ey, ok := y.(*typed.TRecord); ok {
				ex := x.(*typed.TRecord)
				for nx, fx := range ex.Fields {
					if fy, hasField := ey.Fields[nx]; !hasField && !ey.MayHaveMoreFields {
						return common.Error{
							Extra:   append(loc, x.GetLocation(), y.GetLocation()),
							Message: fmt.Sprintf("record missing field `%s`", nx),
						}
					} else if hasField {
						err := unify(fy, fx, append(loc, fy.GetLocation(), fx.GetLocation()), subst)
						if err != nil {
							return err
						}
					}
				}
				return nil
			}
			break
		}
	case *typed.TTuple:
		{
			if ey, ok := y.(*typed.TTuple); ok {
				ex := x.(*typed.TTuple)
				if len(ex.Items) != len(ey.Items) {
					return common.Error{
						Extra:   []ast.Location{ex.Location, ey.Location},
						Message: "tuple sizes mismatch"}
				}
				for i, p := range ex.Items {
					err := unify(p, ey.Items[i], append(loc, p.GetLocation(), ey.Items[i].GetLocation()), subst)
					if err != nil {
						return err
					}
				}
				return nil
			}
			break
		}
	case *typed.TNative:
		{
			if ey, ok := y.(*typed.TNative); ok {
				ex := x.(*typed.TNative)
				if ex.Name == ey.Name {
					if len(ex.Args) == len(ey.Args) {
						for i, p := range ex.Args {
							err := unify(p, ey.Args[i], append(loc, p.GetLocation(), ey.Args[i].GetLocation()), subst)
							if err != nil {
								return err
							}
						}
					}
					return nil
				} else if ex.Name == common.Number {
					if ey.Name == common.NarCoreMathInt || ey.Name == common.NarCoreMathFloat {
						ex.Name = ey.Name
						return nil
					}
				} else if ey.Name == common.Number {
					if ex.Name == common.NarCoreMathInt || ex.Name == common.NarCoreMathFloat {
						ey.Name = ex.Name
						return nil
					}
				}
			}
			break
		}
	case *typed.TData:
		{
			if ey, ok := y.(*typed.TData); ok {
				ex := x.(*typed.TData)
				if ex.Name == ey.Name {
					if len(ex.Args) == len(ey.Args) {
						for i, p := range ex.Args {
							err := unify(p, ey.Args[i], append(loc, p.GetLocation(), ey.Args[i].GetLocation()), subst)
							if err != nil {
								return err
							}
						}
					}
					return nil
				} else if ex.Name == common.Number {
					if ey.Name == common.NarCoreMathInt || ey.Name == common.NarCoreMathFloat {
						ex.Name = ey.Name
						return nil
					}
				} else if ey.Name == common.Number {
					if ex.Name == common.NarCoreMathInt || ex.Name == common.NarCoreMathFloat {
						ey.Name = ex.Name
						return nil
					}
				}
			}
		}
	default:
		return common.NewCompilerError("impossible case")
	}
	if x == nil || y == nil {
		print("todo")
	}
	return common.Error{
		Extra:   append(loc, x.GetLocation(), y.GetLocation()),
		Message: fmt.Sprintf("%v cannot be matched with %v", x, y),
	}
}

func unifyUnbound(v *typed.TUnbound, typ typed.Type, loc []ast.Location, subst map[uint64]typed.Type) error {
	if x, ok := subst[v.Index]; ok {
		return unify(x, typ, loc, subst)
	} else {
		if y, ok := typ.(*typed.TUnbound); ok {
			if uy, c := subst[y.Index]; c {
				return unify(v, uy, loc, subst)
			}
		}
		occurs, err := OccursCheck(v, typ, subst)
		if err != nil {
			return err
		}
		if occurs {
			ata, err := applyType(v, subst)
			if err != nil {
				return err
			}
			atb, err := applyType(typ, subst)
			if err != nil {
				return err
			}
			return common.Error{
				Extra:   append(loc, v.Location, typ.GetLocation()),
				Message: fmt.Sprintf("ambiguous type: %v vs %v", ata, atb),
			}
		}
	}

	if v.Constraint == common.ConstraintNumber {
		switch typ.(type) {
		case *typed.TNative:
			{
				e := typ.(*typed.TNative)
				if e.Name == common.NarCoreMathInt || e.Name == common.NarCoreMathFloat {
					_, err := applyType(typ, subst)
					if err != nil {
						return err
					}
				} else {
					return common.Error{
						Extra:   append(loc, v.Location, typ.GetLocation()),
						Message: fmt.Sprintf("number constrainted type cannot hold %v", typ),
					}
				}
			}
		case *typed.TUnbound:
			{
				x := typ.(*typed.TUnbound)
				x.Constraint = v.Constraint
				typ = x
				break
			}
		}
	}

	subst[v.Index] = typ
	return nil
}

func OccursCheck(v *typed.TUnbound, typ typed.Type, subst map[uint64]typed.Type) (bool, error) {
	if typesEqual(v, typ, nil) {
		return true, nil
	}
	switch typ.(type) {
	case *typed.TFunc:
		{
			e := typ.(*typed.TFunc)
			x, err := OccursCheck(v, e.Return, subst)
			if err != nil {
				return false, err
			}
			if x {
				return true, nil
			}
			for _, p := range e.Params {
				x, err := OccursCheck(v, p, subst)
				if err != nil {
					return false, err
				}
				if x {
					return true, nil
				}
			}
			break
		}
	case *typed.TRecord:
		{
			e := typ.(*typed.TRecord)
			for _, f := range e.Fields {
				x, err := OccursCheck(v, f, subst)
				if err != nil {
					return false, err
				}
				if x {
					return true, nil
				}
			}
			break
		}
	case *typed.TTuple:
		{
			e := typ.(*typed.TTuple)
			for _, i := range e.Items {
				x, err := OccursCheck(v, i, subst)
				if err != nil {
					return false, err
				}
				if x {
					return true, nil
				}
			}
			break
		}
	case *typed.TNative:
		{
			e := typ.(*typed.TNative)
			for _, a := range e.Args {
				x, err := OccursCheck(v, a, subst)
				if err != nil {
					return false, err
				}
				if x {
					return true, nil
				}
			}
			break
		}
	case *typed.TData:
		{
			e := typ.(*typed.TData)
			for _, a := range e.Args {
				x, err := OccursCheck(v, a, subst)
				if err != nil {
					return false, err
				}
				if x {
					return true, nil
				}
			}
		}
	case *typed.TUnbound:
		{
			if c, ok := subst[typ.(*typed.TUnbound).Index]; ok {
				return OccursCheck(v, c, subst)
			}
			break
		}
	default:
		return false, common.NewCompilerError("impossible case")
	}
	return false, nil
}

func applyDefinition(td *typed.Definition, subst map[uint64]typed.Type) (*typed.Definition, error) {
	var err error
	td.Params, err = common.MapError(func(p typed.Pattern) (typed.Pattern, error) {
		return applyPattern(p, subst)
	}, td.Params)
	if err != nil {
		return nil, err
	}
	td.Expression, err = applyExpression(td.Expression, subst)
	if err != nil {
		return nil, err
	}
	return td, nil
}

func applyType(t typed.Type, subst map[uint64]typed.Type) (typed.Type, error) {
	apply := func(x typed.Type) (typed.Type, error) {
		return applyType(x, subst)
	}

	switch t.(type) {
	case *typed.TFunc:
		{
			e := t.(*typed.TFunc)
			params, err := common.MapError(apply, e.Params)
			if err != nil {
				return nil, err
			}
			ret, err := applyType(e.Return, subst)
			if err != nil {
				return nil, err
			}
			t = &typed.TFunc{
				Location: e.Location,
				Params:   params,
				Return:   ret,
			}
			break
		}
	case *typed.TRecord:
		{
			e := t.(*typed.TRecord)
			fields := map[ast.Identifier]typed.Type{}
			for n, x := range e.Fields {
				var err error
				fields[n], err = apply(x)
				if err != nil {
					return nil, err
				}
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
			items, err := common.MapError(apply, e.Items)
			if err != nil {
				return nil, err
			}
			t = &typed.TTuple{
				Location: e.Location,
				Items:    items,
			}
			break
		}
	case *typed.TNative:
		{
			e := t.(*typed.TNative)
			args, err := common.MapError(apply, e.Args)
			if err != nil {
				return nil, err
			}
			t = &typed.TNative{
				Location: e.Location,
				Name:     e.Name,
				Args:     args,
			}
			break
		}
	case *typed.TData:
		{
			e := t.(*typed.TData)
			args, err := common.MapError(apply, e.Args)
			if err != nil {
				return nil, err
			}
			t = &typed.TData{
				Location: e.Location,
				Name:     e.Name,
				Args:     args,
				Options:  e.Options,
			}
		}
	case *typed.TUnbound:
		{
			e := t.(*typed.TUnbound)
			if x, ok := subst[e.Index]; ok {
				var err error
				if e.Constraint == common.ConstraintNumber {
					isNum := false
					switch x.(type) {
					case *typed.TNative:
						e2 := x.(*typed.TNative)
						if e2.Name == common.NarCoreMathInt || e2.Name == common.NarCoreMathFloat {
							isNum = true
						}
					case *typed.TUnbound:
						e2 := x.(*typed.TUnbound)
						e2.Constraint = e.Constraint
						isNum = true
					}
					if !isNum {
						return nil, common.Error{
							Location: e.GetLocation(),
							Message:  fmt.Sprintf("number constrainted type cannot hold %v", x),
						}
					}
				}

				t, err = apply(x)
				if err != nil {
					return nil, err
				}
			}
			break
		}
	default:
		return nil, common.NewCompilerError("impossible case")
	}
	return t, nil
}

func applyPattern(pattern typed.Pattern, subst map[uint64]typed.Type) (typed.Pattern, error) {
	apply := func(x typed.Pattern) (typed.Pattern, error) {
		return applyPattern(x, subst)
	}
	switch pattern.(type) {
	case *typed.PAlias:
		{
			e := pattern.(*typed.PAlias)
			type_, err := applyType(e.Type, subst)
			if err != nil {
				return nil, err
			}
			nested, err := apply(e.Nested)
			if err != nil {
				return nil, err
			}
			pattern = &typed.PAlias{
				Location: e.Location,
				Type:     type_,
				Alias:    e.Alias,
				Nested:   nested,
			}
			break
		}
	case *typed.PAny:
		{
			e := pattern.(*typed.PAny)
			type_, err := applyType(e.Type, subst)
			if err != nil {
				return nil, err
			}
			pattern = &typed.PAny{
				Location: e.Location,
				Type:     type_,
			}
			break
		}
	case *typed.PCons:
		{
			e := pattern.(*typed.PCons)
			type_, err := applyType(e.Type, subst)
			if err != nil {
				return nil, err
			}
			head, err := apply(e.Head)
			if err != nil {
				return nil, err
			}
			tail, err := apply(e.Tail)
			if err != nil {
				return nil, err
			}
			pattern = &typed.PCons{
				Location: e.Location,
				Type:     type_,
				Head:     head,
				Tail:     tail,
			}
			break
		}
	case *typed.PConst:
		{
			e := pattern.(*typed.PConst)
			type_, err := applyType(e.Type, subst)
			if err != nil {
				return nil, err
			}
			pattern = &typed.PConst{
				Location: e.Location,
				Type:     type_,
				Value:    e.Value,
			}
			break
		}
	case *typed.PDataOption:
		{
			e := pattern.(*typed.PDataOption)
			type_, err := applyType(e.Type, subst)
			if err != nil {
				return nil, err
			}
			args, err := common.MapError(apply, e.Args)
			if err != nil {
				return nil, err
			}
			pattern = &typed.PDataOption{
				Location:   e.Location,
				Type:       type_,
				DataName:   e.DataName,
				OptionName: e.OptionName,
				Definition: e.Definition,
				Args:       args,
			}
			break
		}
	case *typed.PList:
		{
			e := pattern.(*typed.PList)
			type_, err := applyType(e.Type, subst)
			if err != nil {
				return nil, err
			}
			items, err := common.MapError(apply, e.Items)
			if err != nil {
				return nil, err
			}
			pattern = &typed.PList{
				Location: e.Location,
				Type:     type_,
				Items:    items,
			}
			break
		}
	case *typed.PNamed:
		{
			e := pattern.(*typed.PNamed)
			type_, err := applyType(e.Type, subst)
			if err != nil {
				return nil, err
			}
			pattern = &typed.PNamed{
				Location: ast.Location{},
				Type:     type_,
				Name:     e.Name,
			}
			break
		}
	case *typed.PRecord:
		{
			e := pattern.(*typed.PRecord)
			type_, err := applyType(e.Type, subst)
			if err != nil {
				return nil, err
			}
			fields, err := common.MapError(func(f typed.PRecordField) (typed.PRecordField, error) {
				type_, err := applyType(f.Type, subst)
				if err != nil {
					return typed.PRecordField{}, err
				}
				return typed.PRecordField{
					Location: f.Location,
					Name:     f.Name,
					Type:     type_,
				}, nil
			}, e.Fields)
			if err != nil {
				return nil, err
			}
			pattern = &typed.PRecord{
				Location: e.Location,
				Type:     type_,
				Fields:   fields,
			}
			break
		}
	case *typed.PTuple:
		{
			e := pattern.(*typed.PTuple)
			type_, err := applyType(e.Type, subst)
			if err != nil {
				return nil, err
			}
			items, err := common.MapError(apply, e.Items)
			if err != nil {
				return nil, err
			}
			pattern = &typed.PTuple{
				Location: e.Location,
				Type:     type_,
				Items:    items,
			}
			break
		}
	default:
		return nil, common.NewCompilerError("impossible case")
	}
	return pattern, nil
}

func applyExpression(expr typed.Expression, subst map[uint64]typed.Type) (typed.Expression, error) {
	if expr == nil {
		return nil, nil
	}

	apply := func(x typed.Expression) (typed.Expression, error) {
		return applyExpression(x, subst)
	}
	switch expr.(type) {
	case *typed.Access:
		{
			e := expr.(*typed.Access)
			type_, err := applyType(e.Type, subst)
			if err != nil {
				return nil, err
			}
			record, err := apply(e.Record)
			if err != nil {
				return nil, err
			}
			expr = &typed.Access{
				Location:  e.Location,
				Type:      type_,
				FieldName: e.FieldName,
				Record:    record,
			}
			break
		}
	case *typed.Apply:
		{
			e := expr.(*typed.Apply)
			type_, err := applyType(e.Type, subst)
			if err != nil {
				return nil, err
			}
			fn, err := apply(e.Func)
			if err != nil {
				return nil, err
			}
			args, err := common.MapError(apply, e.Args)
			if err != nil {
				return nil, err
			}
			expr = &typed.Apply{
				Location: e.Location,
				Type:     type_,
				Func:     fn,
				Args:     args,
			}
			break
		}
	case *typed.Const:
		{
			e := expr.(*typed.Const)
			type_, err := applyType(e.Type, subst)
			if err != nil {
				return nil, err
			}
			expr = &typed.Const{
				Location: e.Location,
				Type:     type_,
				Value:    e.Value,
			}
			break
		}
	case *typed.Let:
		{
			e := expr.(*typed.Let)
			type_, err := applyType(e.Type, subst)
			if err != nil {
				return nil, err
			}
			pattern, err := applyPattern(e.Pattern, subst)
			if err != nil {
				return nil, err
			}
			value, err := apply(e.Value)
			if err != nil {
				return nil, err
			}
			body, err := apply(e.Body)
			if err != nil {
				return nil, err
			}
			expr = &typed.Let{
				Location: e.Location,
				Type:     type_,
				Pattern:  pattern,
				Value:    value,
				Body:     body,
			}
			break
		}
	case *typed.List:
		{
			e := expr.(*typed.List)
			type_, err := applyType(e.Type, subst)
			if err != nil {
				return nil, err
			}
			items, err := common.MapError(apply, e.Items)
			if err != nil {
				return nil, err
			}
			expr = &typed.List{
				Location: e.Location,
				Type:     type_,
				Items:    items,
			}
			break
		}
	case *typed.Record:
		{
			e := expr.(*typed.Record)
			type_, err := applyType(e.Type, subst)
			if err != nil {
				return nil, err
			}
			fields, err := common.MapError(func(f typed.RecordField) (typed.RecordField, error) {
				type_, err := applyType(f.Type, subst)
				if err != nil {
					return typed.RecordField{}, err
				}
				value, err := apply(f.Value)
				if err != nil {
					return typed.RecordField{}, err
				}
				return typed.RecordField{
					Location: f.Location,
					Type:     type_,
					Name:     f.Name,
					Value:    value,
				}, nil
			}, e.Fields)
			if err != nil {
				return nil, err
			}
			expr = &typed.Record{
				Location: e.Location,
				Type:     type_,
				Fields:   fields,
			}
			break
		}
	case *typed.Select:
		{
			e := expr.(*typed.Select)
			type_, err := applyType(e.Type, subst)
			if err != nil {
				return nil, err
			}
			condition, err := apply(e.Condition)
			if err != nil {
				return nil, err
			}
			cases, err := common.MapError(func(c typed.SelectCase) (typed.SelectCase, error) {
				type_, err := applyType(c.Type, subst)
				if err != nil {
					return typed.SelectCase{}, err
				}
				pattern, err := applyPattern(c.Pattern, subst)
				if err != nil {
					return typed.SelectCase{}, err
				}
				expression, err := apply(c.Expression)
				if err != nil {
					return typed.SelectCase{}, err
				}
				return typed.SelectCase{
					Location:   c.Location,
					Type:       type_,
					Pattern:    pattern,
					Expression: expression,
				}, nil
			}, e.Cases)
			if err != nil {
				return nil, err
			}
			expr = &typed.Select{
				Location:  e.Location,
				Type:      type_,
				Condition: condition,
				Cases:     cases,
			}
			break
		}
	case *typed.Tuple:
		{
			e := expr.(*typed.Tuple)
			type_, err := applyType(e.Type, subst)
			if err != nil {
				return nil, err
			}
			items, err := common.MapError(apply, e.Items)
			if err != nil {
				return nil, err
			}
			expr = &typed.Tuple{
				Location: e.Location,
				Type:     type_,
				Items:    items,
			}

			break
		}
	case *typed.UpdateLocal:
		{
			e := expr.(*typed.UpdateLocal)
			type_, err := applyType(e.Type, subst)
			if err != nil {
				return nil, err
			}
			fields, err := common.MapError(func(f typed.RecordField) (typed.RecordField, error) {
				type_, err := applyType(f.Type, subst)
				if err != nil {
					return typed.RecordField{}, err
				}
				value, err := apply(f.Value)
				if err != nil {
					return typed.RecordField{}, err
				}
				return typed.RecordField{
					Location: f.Location,
					Type:     type_,
					Name:     f.Name,
					Value:    value,
				}, nil
			}, e.Fields)
			if err != nil {
				return nil, err
			}
			expr = &typed.UpdateLocal{
				Location:   e.Location,
				Type:       type_,
				RecordName: e.RecordName,
				Fields:     fields,
			}
			break
		}
	case *typed.UpdateGlobal:
		{
			e := expr.(*typed.UpdateGlobal)
			type_, err := applyType(e.Type, subst)
			if err != nil {
				return nil, err
			}
			fields, err := common.MapError(func(f typed.RecordField) (typed.RecordField, error) {
				type_, err := applyType(f.Type, subst)
				if err != nil {
					return typed.RecordField{}, err
				}
				value, err := apply(f.Value)
				if err != nil {
					return typed.RecordField{}, err
				}
				return typed.RecordField{
					Location: f.Location,
					Type:     type_,
					Name:     f.Name,
					Value:    value,
				}, nil
			}, e.Fields)
			if err != nil {
				return nil, err
			}
			expr = &typed.UpdateGlobal{
				Location:       e.Location,
				Type:           type_,
				ModuleName:     e.ModuleName,
				DefinitionName: e.DefinitionName,
				Definition:     e.Definition,
				Fields:         fields,
			}
			break
		}
	case *typed.Constructor:
		{
			e := expr.(*typed.Constructor)
			type_, err := applyType(e.Type, subst)
			if err != nil {
				return nil, err
			}
			args, err := common.MapError(apply, e.Args)
			if err != nil {
				return nil, err
			}
			expr = &typed.Constructor{
				Location:   e.Location,
				Type:       type_,
				DataName:   e.DataName,
				OptionName: e.OptionName,
				Args:       args,
			}

			break
		}
	case *typed.NativeCall:
		{
			e := expr.(*typed.NativeCall)
			type_, err := applyType(e.Type, subst)
			if err != nil {
				return nil, err
			}
			args, err := common.MapError(apply, e.Args)
			if err != nil {
				return nil, err
			}
			expr = &typed.NativeCall{
				Location: e.Location,
				Type:     type_,
				Name:     e.Name,
				Args:     args,
			}
			break
		}

	case *typed.Local:
		{
			e := expr.(*typed.Local)
			type_, err := applyType(e.Type, subst)
			if err != nil {
				return nil, err
			}
			expr = &typed.Local{
				Location: e.Location,
				Type:     type_,
				Name:     e.Name,
			}
			break
		}
	case *typed.Global:
		{
			e := expr.(*typed.Global)
			type_, err := applyType(e.Type, subst)
			if err != nil {
				return nil, err
			}
			expr = &typed.Global{
				Location:       e.Location,
				Type:           type_,
				ModuleName:     e.ModuleName,
				DefinitionName: e.DefinitionName,
				Definition:     e.Definition,
			}
			break
		}
	default:
		return nil, common.NewCompilerError("impossible case")
	}
	return expr, nil
}

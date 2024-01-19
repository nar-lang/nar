package processors

import (
	"fmt"
	"maps"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/ast/parsed"
	"nar-compiler/internal/pkg/common"
	"slices"
	"strings"
	"unicode"
)

var lastDefinitionId = uint64(0)
var lastLambdaId = uint64(0)

type namedTypeMap map[ast.FullIdentifier]*normalized.TPlaceholder

func PreNormalize(
	moduleName ast.QualifiedIdentifier,
	modules map[ast.QualifiedIdentifier]*parsed.Module,
) (errors []error) {
	m, ok := modules[moduleName]
	if !ok {
		return
	}

	flattenDataTypes(m)
	return unwrapImports(m, modules)
}

func Normalize(
	moduleName ast.QualifiedIdentifier,
	modules map[ast.QualifiedIdentifier]*parsed.Module,
	normalizedModules map[ast.QualifiedIdentifier]*normalized.Module,
) (errors []error) {
	if _, ok := normalizedModules[moduleName]; ok {
		return
	}

	m, ok := modules[moduleName]
	if !ok {
		errors = []error{common.Error{Location: m.Location, Message: fmt.Sprintf("module `%s` not found", moduleName)}}
		return
	}

	o := &normalized.Module{
		Name:         m.Name,
		Location:     m.Location,
		Dependencies: map[ast.QualifiedIdentifier][]ast.Identifier{},
	}

	lastLambdaId = 0

	for _, def := range m.Definitions {
		nDef, params, err := normalizeDefinition(modules, m, def, o)
		if err != nil {
			errors = append(errors, err)
		}
		if nDef.Expression != nil {
			nDef.Expression, err = flattenLambdas(nDef.Name, nDef.Expression, o, params)
			if err != nil {
				errors = append(errors, err)
			}
		}
		o.Definitions = append(o.Definitions, nDef)
	}

	normalizedModules[m.Name] = o

	for modName := range o.Dependencies {
		if err := Normalize(modName, modules, normalizedModules); err != nil {
			errors = append(errors, err...)
		}
	}

	return
}

func extractLocals(pattern normalized.Pattern, locals map[ast.Identifier]normalized.Pattern) error {
	switch pattern.(type) {
	case *normalized.PAlias:
		{
			e := pattern.(*normalized.PAlias)
			locals[e.Alias] = pattern
			if err := extractLocals(e.Nested, locals); err != nil {
				return err
			}
			break
		}
	case *normalized.PAny:
		{
			break
		}
	case *normalized.PCons:
		{
			e := pattern.(*normalized.PCons)
			if err := extractLocals(e.Head, locals); err != nil {
				return err
			}
			if err := extractLocals(e.Tail, locals); err != nil {
				return err
			}
			break
		}
	case *normalized.PConst:
		{
			break
		}
	case *normalized.PDataOption:
		{
			e := pattern.(*normalized.PDataOption)
			for _, v := range e.Values {
				if err := extractLocals(v, locals); err != nil {
					return err
				}
			}
			break
		}
	case *normalized.PList:
		{
			e := pattern.(*normalized.PList)
			for _, v := range e.Items {
				if err := extractLocals(v, locals); err != nil {
					return err
				}
			}
			break
		}
	case *normalized.PNamed:
		{
			e := pattern.(*normalized.PNamed)
			locals[e.Name] = pattern
			break
		}
	case *normalized.PRecord:
		{
			e := pattern.(*normalized.PRecord)
			for _, v := range e.Fields {
				locals[v.Name] = pattern
			}
			break
		}
	case *normalized.PTuple:
		{
			e := pattern.(*normalized.PTuple)
			for _, v := range e.Items {
				if err := extractLocals(v, locals); err != nil {
					return err
				}
			}
			break
		}
	default:
		return common.NewCompilerError("impossible case")
	}
	return nil
}

func extractLambda(
	loc ast.Location, parentName ast.Identifier, params []normalized.Pattern, body normalized.Expression,
	m *normalized.Module, locals map[ast.Identifier]normalized.Pattern,
	name ast.Identifier,
) (def *normalized.Definition, usedLocals []ast.Identifier, replacement normalized.Expression, errOut error) {
	lastLambdaId++
	lambdaName := ast.Identifier(fmt.Sprintf("_lmbd_%v_%d_%s", parentName, lastLambdaId, name))
	paramNames, err := extractParamNames(params)
	if err != nil {
		return nil, nil, nil, err
	}
	usedLocals, err = extractUsedLocals(body, locals, paramNames)
	if err != nil {
		return nil, nil, nil, err
	}

	lastDefinitionId++
	def = &normalized.Definition{
		Id:   lastDefinitionId,
		Name: lambdaName,
		Params: append(
			common.Map(func(x ast.Identifier) normalized.Pattern {
				return &normalized.PNamed{
					PatternBase: &normalized.PatternBase{Location: loc},
					Name:        x,
				}
			}, usedLocals),
			params...),
		Expression: body,
		Location:   loc,
		Hidden:     true,
	}
	m.Definitions = append(m.Definitions, def)

	replacement = normalized.Global{
		ExpressionBase: &normalized.ExpressionBase{Location: loc},
		ModuleName:     m.Name,
		DefinitionName: def.Name,
	}
	body.GetPredecessor().SetSuccessor(replacement)

	if len(usedLocals) > 0 {
		replacement = normalized.Apply{
			ExpressionBase: &normalized.ExpressionBase{Location: loc},
			Func:           replacement,
			Args: common.Map(func(x ast.Identifier) normalized.Expression {
				return normalized.Local{
					ExpressionBase: &normalized.ExpressionBase{Location: loc},
					Name:           x,
				}
			}, usedLocals),
		}
		body.GetPredecessor().SetSuccessor(replacement)
	}

	return
}

func extractParamNames(params []normalized.Pattern) (map[ast.Identifier]normalized.Pattern, error) {
	paramNames := map[ast.Identifier]normalized.Pattern{}
	for _, p := range params {
		if err := extractLocals(p, paramNames); err != nil {
			return nil, err
		}
	}
	return paramNames, nil
}

func flattenLambdas(
	parentName ast.Identifier,
	expr normalized.Expression, m *normalized.Module, locals map[ast.Identifier]normalized.Pattern,
) (normalized.Expression, error) {
	var err error
	switch expr.(type) {
	case normalized.Lambda:
		{
			e := expr.(normalized.Lambda)
			def, _, replacement, err := extractLambda(e.Location, parentName, e.Params, e.Body, m, locals, "")
			if err != nil {
				return nil, err
			}
			paramNames, err := extractParamNames(def.Params)
			if err != nil {
				return nil, err
			}
			def.Expression, err = flattenLambdas(def.Name, def.Expression, m, paramNames)
			if err != nil {
				return nil, err
			}
			return replacement, nil
		}
	case normalized.LetDef:
		{
			e := expr.(normalized.LetDef)
			def, usedLocals, replacement, err := extractLambda(
				e.Location, parentName, e.Params, e.Body, m, locals, e.Name)
			if err != nil {
				return nil, err
			}

			if len(usedLocals) > 0 {
				replName := ast.Identifier(fmt.Sprintf("_lambda_closue_%d", lastLambdaId))
				replaceMap := map[ast.Identifier]normalized.Expression{}
				r := normalized.Local{
					ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
					Name:           replName,
				}
				replaceMap[e.Name] = r
				e.GetPredecessor().SetSuccessor(r)

				let := normalized.LetMatch{
					ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
					Pattern: &normalized.PNamed{
						PatternBase: &normalized.PatternBase{Location: e.Location},
						Name:        replName,
					},
					Value:  replacement,
					Nested: def.Expression,
				}
				e.GetPredecessor().SetSuccessor(let)
				def.Expression = let
				def.Expression, err = replaceLocals(def.Expression, replaceMap)
				if err != nil {
					return nil, err
				}
				paramNames, err := extractParamNames(def.Params)
				if err != nil {
					return nil, err
				}
				def.Expression, err = flattenLambdas(def.Name, def.Expression, m, paramNames)
				if err != nil {
					return nil, err
				}

				let.Nested, err = replaceLocals(e.Nested, replaceMap)
				if err != nil {
					return nil, err
				}
				let.Nested, err = flattenLambdas(parentName, let.Nested, m, locals)
				if err != nil {
					return nil, err
				}
				return let, nil
			} else {
				replaceMap := map[ast.Identifier]normalized.Expression{}
				replaceMap[e.Name] = replacement

				def.Expression, err = replaceLocals(def.Expression, replaceMap)
				if err != nil {
					return nil, err
				}
				paramNames, err := extractParamNames(def.Params)
				if err != nil {
					return nil, err
				}
				def.Expression, err = flattenLambdas(def.Name, def.Expression, m, paramNames)
				if err != nil {
					return nil, err
				}

				replacedLocals, err := replaceLocals(e.Nested, replaceMap)
				if err != nil {
					return nil, err
				}
				return flattenLambdas(parentName, replacedLocals, m, locals)
			}
		}
	case normalized.LetMatch:
		{
			e := expr.(normalized.LetMatch)
			innerLocals := maps.Clone(locals)
			err := extractLocals(e.Pattern, innerLocals)
			if err != nil {
				return nil, err
			}
			e.Value, err = flattenLambdas(parentName, e.Value, m, innerLocals)
			if err != nil {
				return nil, err
			}
			e.Nested, err = flattenLambdas(parentName, e.Nested, m, innerLocals)
			if err != nil {
				return nil, err
			}
			return e, nil
		}
	case normalized.Access:
		{
			e := expr.(normalized.Access)
			e.Record, err = flattenLambdas(parentName, e.Record, m, locals)
			if err != nil {
				return nil, err
			}
			return e, nil
		}
	case normalized.Apply:
		{
			e := expr.(normalized.Apply)
			e.Func, err = flattenLambdas(parentName, e.Func, m, locals)
			if err != nil {
				return nil, err
			}
			for i, a := range e.Args {
				e.Args[i], err = flattenLambdas(parentName, a, m, locals)
				if err != nil {
					return nil, err
				}
			}
			return e, nil
		}
	case normalized.List:
		{
			e := expr.(normalized.List)
			for i, a := range e.Items {
				e.Items[i], err = flattenLambdas(parentName, a, m, locals)
				if err != nil {
					return nil, err
				}
			}
			return e, nil
		}
	case normalized.Record:
		{
			e := expr.(normalized.Record)
			for i, a := range e.Fields {
				e.Fields[i].Value, err = flattenLambdas(parentName, a.Value, m, locals)
				if err != nil {
					return nil, err
				}
			}
			return e, nil
		}
	case normalized.Select:
		{
			e := expr.(normalized.Select)
			e.Condition, err = flattenLambdas(parentName, e.Condition, m, locals)
			if err != nil {
				return nil, err
			}
			for i, a := range e.Cases {
				innerLocals := maps.Clone(locals)
				err = extractLocals(a.Pattern, innerLocals)
				if err != nil {
					return nil, err
				}
				e.Cases[i].Expression, err = flattenLambdas(parentName, a.Expression, m, innerLocals)
				if err != nil {
					return nil, err
				}
			}
			return e, nil
		}
	case normalized.Tuple:
		{
			e := expr.(normalized.Tuple)
			for i, a := range e.Items {
				e.Items[i], err = flattenLambdas(parentName, a, m, locals)
				if err != nil {
					return nil, err
				}
			}
			return e, nil
		}
	case normalized.UpdateLocal:
		{
			e := expr.(normalized.UpdateLocal)
			for i, a := range e.Fields {
				e.Fields[i].Value, err = flattenLambdas(parentName, a.Value, m, locals)
				if err != nil {
					return nil, err
				}
			}
			return e, nil
		}
	case normalized.UpdateGlobal:
		{
			e := expr.(normalized.UpdateGlobal)
			for i, a := range e.Fields {
				e.Fields[i].Value, err = flattenLambdas(parentName, a.Value, m, locals)
				if err != nil {
					return nil, err
				}
			}
			return e, nil
		}
	case normalized.Constructor:
		{
			e := expr.(normalized.Constructor)
			for i, a := range e.Args {
				e.Args[i], err = flattenLambdas(parentName, a, m, locals)
				if err != nil {
					return nil, err
				}
			}
			return e, nil
		}
	case normalized.NativeCall:
		{
			e := expr.(normalized.NativeCall)
			for i, a := range e.Args {
				e.Args[i], err = flattenLambdas(parentName, a, m, locals)
				if err != nil {
					return nil, err
				}
			}
			return e, nil
		}
	case normalized.Const:
		{
			return expr, nil
		}
	case normalized.Global:
		{
			return expr, nil
		}
	case normalized.Local:
		{
			return expr, nil
		}
	default:
		return nil, common.NewCompilerError("impossible case")
	}
}

func replaceLocals(
	expr normalized.Expression, replace map[ast.Identifier]normalized.Expression,
) (normalized.Expression, error) {
	var err error
	switch expr.(type) {
	case normalized.Lambda:
		{
			e := expr.(normalized.Lambda)
			e.Body, err = replaceLocals(e.Body, replace)
			if err != nil {
				return nil, err
			}
			return e, nil
		}
	case normalized.LetDef:
		{
			e := expr.(normalized.LetDef)
			e.Body, err = replaceLocals(e.Body, replace)
			if err != nil {
				return nil, err
			}
			e.Nested, err = replaceLocals(e.Nested, replace)
			if err != nil {
				return nil, err
			}
			return e, nil
		}
	case normalized.LetMatch:
		{
			e := expr.(normalized.LetMatch)
			e.Value, err = replaceLocals(e.Value, replace)
			if err != nil {
				return nil, err
			}
			e.Nested, err = replaceLocals(e.Nested, replace)
			if err != nil {
				return nil, err
			}
			return e, nil
		}
	case normalized.Access:
		{
			e := expr.(normalized.Access)
			e.Record, err = replaceLocals(e.Record, replace)
			if err != nil {
				return nil, err
			}
			return e, nil
		}
	case normalized.Apply:
		{
			e := expr.(normalized.Apply)
			e.Func, err = replaceLocals(e.Func, replace)
			if err != nil {
				return nil, err
			}
			for i, a := range e.Args {
				e.Args[i], err = replaceLocals(a, replace)
				if err != nil {
					return nil, err
				}
			}
			return e, nil
		}
	case normalized.List:
		{
			e := expr.(normalized.List)
			for i, a := range e.Items {
				e.Items[i], err = replaceLocals(a, replace)
				if err != nil {
					return nil, err
				}
			}
			return e, nil
		}
	case normalized.Record:
		{
			e := expr.(normalized.Record)
			for i, a := range e.Fields {
				e.Fields[i].Value, err = replaceLocals(a.Value, replace)
				if err != nil {
					return nil, err
				}
			}
			return e, nil
		}
	case normalized.Select:
		{
			e := expr.(normalized.Select)
			e.Condition, err = replaceLocals(e.Condition, replace)
			if err != nil {
				return nil, err
			}
			for i, a := range e.Cases {
				e.Cases[i].Expression, err = replaceLocals(a.Expression, replace)
				if err != nil {
					return nil, err
				}
			}
			return e, nil
		}
	case normalized.Tuple:
		{
			e := expr.(normalized.Tuple)
			for i, a := range e.Items {
				e.Items[i], err = replaceLocals(a, replace)
				if err != nil {
					return nil, err
				}
			}
			return e, nil
		}
	case normalized.UpdateLocal:
		{
			e := expr.(normalized.UpdateLocal)
			for i, a := range e.Fields {
				e.Fields[i].Value, err = replaceLocals(a.Value, replace)
				if err != nil {
					return nil, err
				}
			}
			return e, nil
		}
	case normalized.UpdateGlobal:
		{
			e := expr.(normalized.UpdateGlobal)
			for i, a := range e.Fields {
				e.Fields[i].Value, err = replaceLocals(a.Value, replace)
				if err != nil {
					return nil, err
				}
			}
			return e, nil
		}
	case normalized.Constructor:
		{
			e := expr.(normalized.Constructor)
			for i, a := range e.Args {
				e.Args[i], err = replaceLocals(a, replace)
				if err != nil {
					return nil, err
				}
			}
			return e, nil
		}
	case normalized.NativeCall:
		{
			e := expr.(normalized.NativeCall)
			for i, a := range e.Args {
				e.Args[i], err = replaceLocals(a, replace)
				if err != nil {
					return nil, err
				}
			}
			return e, nil
		}
	case normalized.Const:
		{
			return expr, nil
		}
	case normalized.Global:
		{
			return expr, nil
		}
	case normalized.Local:
		{
			e := expr.(normalized.Local)
			if r, ok := replace[e.Name]; ok {
				return r, nil
			}
			return expr, nil
		}
	default:
		return nil, common.NewCompilerError("impossible case")
	}
}

func flattenDataTypes(m *parsed.Module) {
	for _, it := range m.DataTypes {
		typeArgs := common.Map(func(x ast.Identifier) parsed.Type {
			return &parsed.TTypeParameter{
				Location: it.Location,
				Name:     x,
			}
		}, it.Params)

		dataType := &parsed.TData{
			Location: it.Location,
			Name:     common.MakeFullIdentifier(m.Name, it.Name),
			Args:     typeArgs,
			Options: common.Map(func(x parsed.DataTypeOption) parsed.DataOption {
				return parsed.DataOption{
					Name:   x.Name,
					Hidden: x.Hidden,
					Values: x.Values,
				}
			}, it.Options),
		}
		m.Aliases = append(m.Aliases, parsed.Alias{
			Location: it.Location,
			Name:     it.Name,
			Params:   it.Params,
			Type:     dataType,
		})
		for _, option := range it.Options {
			var type_ parsed.Type = dataType
			if len(option.Values) > 0 {
				type_ = &parsed.TFunc{
					Location: it.Location,
					Params:   option.Values,
					Return:   type_,
				}
			}
			var body parsed.Expression = &parsed.Constructor{
				ExpressionBase: &parsed.ExpressionBase{
					Location: option.Location,
				},
				ModuleName: m.Name,
				DataName:   it.Name,
				OptionName: option.Name,
				Args: common.Map(
					func(i int) parsed.Expression {
						return &parsed.Var{
							ExpressionBase: &parsed.ExpressionBase{
								Location: option.Location,
							},
							Name: ast.QualifiedIdentifier(fmt.Sprintf("p%d", i)),
						}
					},
					common.Range(0, len(option.Values)),
				),
			}

			params := common.Map(
				func(i int) parsed.Pattern {
					return &parsed.PNamed{Location: option.Location, Name: ast.Identifier(fmt.Sprintf("p%d", i))}
				},
				common.Range(0, len(option.Values)),
			)

			m.Definitions = append(m.Definitions, &parsed.Definition{
				Location:   option.Location,
				Hidden:     option.Hidden || it.Hidden,
				Name:       option.Name,
				Params:     params,
				Expression: body,
				Type:       type_,
			})
		}
	}
}

func unwrapImports(module *parsed.Module, modules map[ast.QualifiedIdentifier]*parsed.Module) (errors []error) {
	for i, imp := range module.Imports {
		m, ok := modules[imp.ModuleIdentifier]
		if !ok {
			errors = append(errors, common.Error{
				Location: imp.Location,
				Message:  fmt.Sprintf("module `%s` not found", imp.ModuleIdentifier),
			})
			continue
		}
		modName := m.Name
		if imp.Alias != nil {
			modName = ast.QualifiedIdentifier(*imp.Alias)
		}
		shortModName := ast.QualifiedIdentifier("")
		lastDotIndex := strings.LastIndex(string(modName), ".")
		if lastDotIndex >= 0 {
			shortModName = modName[lastDotIndex+1:]
		}

		var exp []string
		expose := func(n string, exn string) {
			if imp.ExposingAll || slices.Contains(imp.Exposing, exn) {
				exp = append(exp, n)
			}
			exp = append(exp, fmt.Sprintf("%s.%s", modName, n))
			if shortModName != "" {
				exp = append(exp, fmt.Sprintf("%s.%s", shortModName, n))
			}
		}

		for _, d := range m.Definitions {
			if !d.Hidden {
				expose(string(d.Name), string(d.Name))
			}
		}

		for _, a := range m.Aliases {
			if !a.Hidden {
				expose(string(a.Name), string(a.Name))
				if dt, ok := a.Type.(*parsed.TData); ok {
					for _, v := range dt.Options {
						if !v.Hidden {
							expose(string(v.Name), string(a.Name))
						}
					}
				}
			}
		}

		for _, a := range m.InfixFns {
			if !a.Hidden {
				expose(string(a.Name), string(a.Name))
			}
		}
		imp.Exposing = exp
		module.Imports[i] = imp
	}
	return
}

func normalizeDefinition(
	modules map[ast.QualifiedIdentifier]*parsed.Module, module *parsed.Module, def *parsed.Definition,
	normalizedModule *normalized.Module,
) (o *normalized.Definition, params map[ast.Identifier]normalized.Pattern, err error) {
	lastDefinitionId++
	o = &normalized.Definition{
		Id:       lastDefinitionId,
		Name:     def.Name,
		Location: def.Location,
		Hidden:   def.Hidden,
	}
	params = map[ast.Identifier]normalized.Pattern{}
	o.Params, err = common.MapError(func(x parsed.Pattern) (normalized.Pattern, error) {
		return normalizePattern(params, modules, module, x, normalizedModule)
	}, def.Params)
	if err != nil {
		return
	}
	locals := maps.Clone(params)
	if def.Expression != nil {
		o.Expression, err = normalizeExpression(locals, modules, module, def.Expression, normalizedModule)
		if err != nil {
			return
		}
	}
	o.Type, err = normalizeType(modules, module, nil, def.Type, nil)
	if err != nil {
		return
	}
	return
}

func normalizePattern(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*parsed.Module, module *parsed.Module,
	pattern parsed.Pattern,
	normalizedModule *normalized.Module,
) (normalized.Pattern, error) {
	normalize := func(p parsed.Pattern) (normalized.Pattern, error) {
		return normalizePattern(locals, modules, module, p, normalizedModule)
	}

	switch pattern.(type) {
	case *parsed.PAlias:
		{
			e := pattern.(*parsed.PAlias)
			np := &normalized.PAlias{
				PatternBase: &normalized.PatternBase{Location: e.Location},
				Alias:       e.Alias,
			}
			locals[e.Alias] = np
			var err error
			np.Nested, err = normalize(e.Nested)
			if err != nil {
				return nil, err
			}
			np.Type, err = normalizeType(modules, module, nil, e.Type, nil)
			if err != nil {
				return nil, err
			}
			return np, nil
		}
	case *parsed.PAny:
		{
			e := pattern.(*parsed.PAny)
			type_, err := normalizeType(modules, module, nil, e.Type, nil)
			if err != nil {
				return nil, err
			}
			return &normalized.PAny{
				PatternBase: &normalized.PatternBase{Location: e.Location},
				Type:        type_,
			}, nil
		}
	case *parsed.PCons:
		{
			e := pattern.(*parsed.PCons)
			head, err := normalize(e.Head)
			if err != nil {
				return nil, err
			}
			tail, err := normalize(e.Tail)
			if err != nil {
				return nil, err
			}
			type_, err := normalizeType(modules, module, nil, e.Type, nil)
			if err != nil {
				return nil, err
			}
			return &normalized.PCons{
				PatternBase: &normalized.PatternBase{Location: e.Location},
				Type:        type_,
				Head:        head,
				Tail:        tail,
			}, nil
		}
	case *parsed.PConst:
		{
			e := pattern.(*parsed.PConst)
			type_, err := normalizeType(modules, module, nil, e.Type, nil)
			if err != nil {
				return nil, err
			}
			return &normalized.PConst{
				PatternBase: &normalized.PatternBase{Location: e.Location},
				Type:        type_,
				Value:       e.Value,
			}, nil
		}
	case *parsed.PDataOption:
		{
			e := pattern.(*parsed.PDataOption)
			def, mod, ids := FindParsedDefinition(modules, module, e.Name, normalizedModule)
			if len(ids) == 0 {
				return nil, common.Error{Location: e.Location, Message: "data constructor not found"}
			} else if len(ids) > 1 {
				return nil, common.Error{
					Location: e.Location,
					Message: fmt.Sprintf(
						"ambiguous data constructor `%s`, it can be one of %s. "+
							"Use import or qualified identifer to clarify which one to use",
						e.Name, common.Join(ids, ", ")),
				}
			}
			values, err := common.MapError(normalize, e.Values)
			if err != nil {
				return nil, err
			}
			type_, err := normalizeType(modules, module, nil, e.Type, nil)
			if err != nil {
				return nil, err
			}
			return &normalized.PDataOption{
				PatternBase:    &normalized.PatternBase{Location: e.Location},
				Type:           type_,
				ModuleName:     mod.Name,
				DefinitionName: def.Name,
				Values:         values,
			}, nil
		}
	case *parsed.PList:
		{
			e := pattern.(*parsed.PList)
			items, err := common.MapError(normalize, e.Items)
			if err != nil {
				return nil, err
			}
			type_, err := normalizeType(modules, module, nil, e.Type, nil)
			if err != nil {
				return nil, err
			}
			return &normalized.PList{
				PatternBase: &normalized.PatternBase{Location: e.Location},
				Type:        type_,
				Items:       items,
			}, nil
		}
	case *parsed.PNamed:
		{
			e := pattern.(*parsed.PNamed)
			np := &normalized.PNamed{
				PatternBase: &normalized.PatternBase{Location: e.Location},
				Name:        e.Name,
			}
			locals[e.Name] = np
			var err error
			np.Type, err = normalizeType(modules, module, nil, e.Type, nil)
			if err != nil {
				return nil, err
			}
			return np, nil
		}
	case *parsed.PRecord:
		{
			e := pattern.(*parsed.PRecord)
			type_, err := normalizeType(modules, module, nil, e.Type, nil)
			if err != nil {
				return nil, err
			}
			return &normalized.PRecord{
				PatternBase: &normalized.PatternBase{Location: e.Location},
				Type:        type_,
				Fields: common.Map(func(x parsed.PRecordField) normalized.PRecordField {
					locals[x.Name] = &normalized.PNamed{
						PatternBase: &normalized.PatternBase{Location: x.Location},
						Name:        x.Name,
					}
					return normalized.PRecordField{Location: x.Location, Name: x.Name}
				}, e.Fields),
			}, nil
		}
	case *parsed.PTuple:
		{
			e := pattern.(*parsed.PTuple)
			items, err := common.MapError(normalize, e.Items)
			if err != nil {
				return nil, err
			}
			type_, err := normalizeType(modules, module, nil, e.Type, nil)
			if err != nil {
				return nil, err
			}
			return &normalized.PTuple{
				PatternBase: &normalized.PatternBase{Location: e.Location},
				Type:        type_,
				Items:       items,
			}, nil
		}
	}
	return nil, common.NewCompilerError("impossible case")
}

func normalizeExpression(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*parsed.Module,
	module *parsed.Module,
	expr parsed.Expression,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := func(e parsed.Expression) (normalized.Expression, error) {
		return normalizeExpression(locals, modules, module, e, normalizedModule)
	}
	newAmbiguousInfixError := func(ids []ast.FullIdentifier, name ast.InfixIdentifier, loc ast.Location) error {
		if len(ids) == 0 {
			return common.Error{
				Location: loc,
				Message:  fmt.Sprintf("infix definition `%s` not found", name),
			}
		} else {
			return common.Error{
				Location: loc,
				Message: fmt.Sprintf(
					"ambiguous infix identifier `%s`, it can be one of %s. Use import to clarify which one to use",
					name, common.Join(ids, ", ")),
			}
		}
	}
	newAmbiguousDefinitionError := func(ids []ast.FullIdentifier, name ast.QualifiedIdentifier, loc ast.Location) error {
		if len(ids) == 0 {
			return common.Error{
				Location: loc,
				Message:  fmt.Sprintf("definition `%s` not found", name),
			}
		} else {
			return common.Error{
				Location: loc,
				Message: fmt.Sprintf(
					"ambiguous identifier `%s`, it can be one of %s. Use import or qualified identifer to clarify which one to use",
					name, common.Join(ids, ", ")),
			}
		}
	}

	switch expr.(type) {
	case *parsed.Access:
		{
			e := expr.(*parsed.Access)
			record, err := normalize(e.Record)
			if err != nil {
				return nil, err
			}
			return e.SetSuccessor(normalized.Access{
				ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
				Record:         record,
				FieldName:      e.FieldName,
			}), nil
		}
	case *parsed.Apply:
		{
			e := expr.(*parsed.Apply)
			fn, err := normalize(e.Func)
			if err != nil {
				return nil, err
			}
			args, err := common.MapError(normalize, e.Args)
			if err != nil {
				return nil, err
			}
			return e.SetSuccessor(normalized.Apply{
				ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
				Func:           fn,
				Args:           args,
			}), nil
		}
	case *parsed.Const:
		{
			e := expr.(*parsed.Const)
			return e.SetSuccessor(normalized.Const{
				ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
				Value:          e.Value,
			}), nil
		}
	case *parsed.Constructor:
		{
			e := expr.(*parsed.Constructor)
			args, err := common.MapError(normalize, e.Args)
			if err != nil {
				return nil, err
			}
			return e.SetSuccessor(normalized.Constructor{
				ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
				ModuleName:     e.ModuleName,
				DataName:       e.DataName,
				OptionName:     e.OptionName,
				Args:           args,
			}), nil
		}
	case *parsed.If:
		{
			e := expr.(*parsed.If)
			boolType := &normalized.TData{
				Location: e.Location,
				Name:     common.NarCoreBasicsBool,
				Options: []normalized.DataOption{
					{Name: common.NarCoreBasicsTrueName},
					{Name: common.NarCoreBasicsFalseName},
				},
			}
			condition, err := normalize(e.Condition)
			if err != nil {
				return nil, err
			}
			positive, err := normalize(e.Positive)
			if err != nil {
				return nil, err
			}
			negative, err := normalize(e.Negative)
			if err != nil {
				return nil, err
			}
			return e.SetSuccessor(normalized.Select{
				ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
				Condition:      condition,
				Cases: []normalized.SelectCase{
					{
						Location: e.Positive.GetLocation(),
						Pattern: &normalized.PDataOption{
							PatternBase:    &normalized.PatternBase{Location: e.Positive.GetLocation()},
							Type:           boolType,
							ModuleName:     common.NarCoreBasicsName,
							DefinitionName: common.NarCoreBasicsTrueName,
						},
						Expression: positive,
					},
					{
						Location: e.Negative.GetLocation(),
						Pattern: &normalized.PDataOption{
							PatternBase:    &normalized.PatternBase{Location: e.Negative.GetLocation()},
							Type:           boolType,
							ModuleName:     common.NarCoreBasicsName,
							DefinitionName: common.NarCoreBasicsFalseName,
						},
						Expression: negative,
					},
				},
			}), nil
		}
	case *parsed.LetMatch:
		{
			e := expr.(*parsed.LetMatch)
			innerLocals := maps.Clone(locals)
			pattern, err := normalizePattern(innerLocals, modules, module, e.Pattern, normalizedModule)
			if err != nil {
				return nil, err
			}
			value, err := normalizeExpression(innerLocals, modules, module, e.Value, normalizedModule)
			if err != nil {
				return nil, err
			}
			nested, err := normalizeExpression(innerLocals, modules, module, e.Nested, normalizedModule)
			if err != nil {
				return nil, err
			}
			return e.SetSuccessor(normalized.LetMatch{
				ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
				Pattern:        pattern,
				Value:          value,
				Nested:         nested,
			}), nil
		}
	case *parsed.LetDef:
		{
			e := expr.(*parsed.LetDef)
			innerLocals := maps.Clone(locals)
			innerLocals[e.Name] = &normalized.PNamed{
				PatternBase: &normalized.PatternBase{Location: e.NameLocation},
				Name:        e.Name,
			}
			params, err := common.MapError(func(x parsed.Pattern) (normalized.Pattern, error) {
				return normalizePattern(innerLocals, modules, module, x, normalizedModule)
			}, e.Params)
			if err != nil {
				return nil, err
			}
			body, err := normalizeExpression(innerLocals, modules, module, e.Body, normalizedModule)
			if err != nil {
				return nil, err
			}
			nested, err := normalizeExpression(innerLocals, modules, module, e.Nested, normalizedModule)
			if err != nil {
				return nil, err
			}
			type_, err := normalizeType(modules, module, nil, e.FnType, nil)
			if err != nil {
				return nil, err
			}
			return e.SetSuccessor(normalized.LetDef{
				ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
				Name:           e.Name,
				Params:         params,
				FnType:         type_,
				Body:           body,
				Nested:         nested,
			}), nil
		}
	case *parsed.List:
		{
			e := expr.(*parsed.List)
			items, err := common.MapError(normalize, e.Items)
			if err != nil {
				return nil, err
			}
			return e.SetSuccessor(normalized.List{
				ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
				Items:          items,
			}), nil
		}
	case *parsed.NativeCall:
		{
			e := expr.(*parsed.NativeCall)
			args, err := common.MapError(normalize, e.Args)
			if err != nil {
				return nil, err
			}
			return e.SetSuccessor(normalized.NativeCall{
				ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
				Name:           e.Name,
				Args:           args,
			}), nil
		}
	case *parsed.Record:
		{
			e := expr.(*parsed.Record)
			fields, err := common.MapError(func(i parsed.RecordField) (normalized.RecordField, error) {
				value, err := normalize(i.Value)
				if err != nil {
					return normalized.RecordField{}, err
				}
				return normalized.RecordField{
					Location: i.Location,
					Name:     i.Name,
					Value:    value,
				}, nil
			}, e.Fields)
			if err != nil {
				return nil, err
			}
			return e.SetSuccessor(normalized.Record{
				ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
				Fields:         fields,
			}), nil
		}
	case *parsed.Select:
		{
			e := expr.(*parsed.Select)
			condition, err := normalize(e.Condition)
			if err != nil {
				return nil, err
			}
			cases, err := common.MapError(func(i parsed.SelectCase) (normalized.SelectCase, error) {
				innerLocals := maps.Clone(locals)
				pattern, err := normalizePattern(innerLocals, modules, module, i.Pattern, normalizedModule)
				if err != nil {
					return normalized.SelectCase{}, err
				}
				expression, err := normalizeExpression(innerLocals, modules, module, i.Expression, normalizedModule)
				if err != nil {
					return normalized.SelectCase{}, err
				}
				return normalized.SelectCase{
					Location:   e.Location,
					Pattern:    pattern,
					Expression: expression,
				}, nil
			}, e.Cases)
			if err != nil {
				return nil, err
			}
			return e.SetSuccessor(normalized.Select{
				ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
				Condition:      condition,
				Cases:          cases,
			}), nil
		}
	case *parsed.Tuple:
		{
			e := expr.(*parsed.Tuple)
			items, err := common.MapError(normalize, e.Items)
			if err != nil {
				return nil, err
			}
			return e.SetSuccessor(normalized.Tuple{
				ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
				Items:          items,
			}), nil
		}
	case *parsed.Update:
		{
			e := expr.(*parsed.Update)
			d, m, ids := FindParsedDefinition(modules, module, e.RecordName, normalizedModule)
			fields, err := common.MapError(func(i parsed.RecordField) (normalized.RecordField, error) {
				value, err := normalize(i.Value)
				if err != nil {
					return normalized.RecordField{}, err
				}
				return normalized.RecordField{
					Location: i.Location,
					Name:     i.Name,
					Value:    value,
				}, nil
			}, e.Fields)
			if err != nil {
				return nil, err
			}

			if len(ids) == 1 {

				return e.SetSuccessor(normalized.UpdateGlobal{
					ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
					ModuleName:     m.Name,
					DefinitionName: d.Name,
					Fields:         fields,
				}), nil
			} else if len(ids) > 1 {
				return nil, newAmbiguousDefinitionError(ids, e.RecordName, e.Location)
			}

			return e.SetSuccessor(normalized.UpdateLocal{
				ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
				RecordName:     ast.Identifier(e.RecordName),
				Fields:         fields,
			}), nil
		}
	case *parsed.Lambda:
		{
			e := expr.(*parsed.Lambda)
			params, err := common.MapError(func(x parsed.Pattern) (normalized.Pattern, error) {
				return normalizePattern(locals, modules, module, x, normalizedModule)
			}, e.Params)
			if err != nil {
				return nil, err
			}
			body, err := normalize(e.Body)
			if err != nil {
				return nil, err
			}
			return e.SetSuccessor(normalized.Lambda{
				ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
				Params:         params,
				Body:           body,
			}), nil
		}
	case *parsed.Accessor:
		{
			e := expr.(*parsed.Accessor)
			return normalize(&parsed.Lambda{
				Params: []parsed.Pattern{&parsed.PNamed{Location: e.Location, Name: "x"}},
				Body: &parsed.Access{
					ExpressionBase: &parsed.ExpressionBase{
						Location: e.Location,
					},
					Record: &parsed.Var{
						ExpressionBase: &parsed.ExpressionBase{
							Location: e.Location,
						},
						Name: "x",
					},
					FieldName: e.FieldName,
				},
			})
		}
	case *parsed.BinOp:
		{
			e := expr.(*parsed.BinOp)
			var output []parsed.BinOpItem
			var operators []parsed.BinOpItem
			for _, o1 := range e.Items {
				if o1.Expression != nil {
					output = append(output, o1)
				} else {
					if infixFn, _, ids := findParsedInfixFn(modules, module, o1.Infix); len(ids) != 1 {
						return nil, newAmbiguousInfixError(ids, o1.Infix, e.Location)
					} else {
						o1.Fn = infixFn
					}

					for i := len(operators) - 1; i >= 0; i-- {
						o2 := operators[i]
						if o2.Fn.Precedence > o1.Fn.Precedence ||
							(o2.Fn.Precedence == o1.Fn.Precedence && o1.Fn.Associativity == parsed.Left) {
							output = append(output, o2)
							operators = operators[:len(operators)-1]
						} else {
							break
						}
					}
					operators = append(operators, o1)
				}
			}
			for i := len(operators) - 1; i >= 0; i-- {
				output = append(output, operators[i])
			}

			var buildTree func() (normalized.Expression, error)
			buildTree = func() (normalized.Expression, error) {
				op := output[len(output)-1].Infix
				output = output[:len(output)-1]

				if infixA, m, ids := findParsedInfixFn(modules, module, op); len(ids) != 1 {
					return nil, newAmbiguousInfixError(ids, op, e.Location)
				} else {
					var left, right normalized.Expression
					var err error
					r := output[len(output)-1]
					if r.Expression != nil {
						right, err = normalize(r.Expression)
						if err != nil {
							return nil, err
						}
						output = output[:len(output)-1]
					} else {
						right, err = buildTree()
						if err != nil {
							return nil, err
						}
					}

					l := output[len(output)-1]
					if l.Expression != nil {
						left, err = normalize(l.Expression)
						if err != nil {
							return nil, err
						}
						output = output[:len(output)-1]
					} else {
						left, err = buildTree()
						if err != nil {
							return nil, err
						}
					}

					return e.SetSuccessor(normalized.Apply{
						ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
						Func: normalized.Global{
							ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
							ModuleName:     m.Name,
							DefinitionName: infixA.Alias,
						},
						Args: []normalized.Expression{left, right},
					}), nil
				}
			}

			return buildTree()
		}
	case *parsed.Negate:
		{
			e := expr.(*parsed.Negate)
			nested, err := normalize(e.Nested)
			if err != nil {
				return nil, err
			}
			return e.SetSuccessor(normalized.Apply{
				ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
				Func: normalized.Global{
					ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
					ModuleName:     common.NarCoreMath,
					DefinitionName: common.NarCoreMathNeg,
				},
				Args: []normalized.Expression{nested},
			}), nil
		}
	case *parsed.Var:
		{
			e := expr.(*parsed.Var)
			if lc, ok := locals[ast.Identifier(e.Name)]; ok {
				return e.SetSuccessor(normalized.Local{
					ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
					Name:           ast.Identifier(e.Name),
					Target:         lc,
				}), nil
			}

			d, m, ids := FindParsedDefinition(modules, module, e.Name, normalizedModule)
			if len(ids) == 1 {
				return e.SetSuccessor(normalized.Global{
					ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
					ModuleName:     m.Name,
					DefinitionName: d.Name,
				}), nil
			} else if len(ids) > 1 {
				return nil, newAmbiguousDefinitionError(ids, e.Name, e.Location)
			}

			parts := strings.Split(string(e.Name), ".")
			if len(parts) > 1 {
				varAccess := parsed.Expression(&parsed.Var{
					ExpressionBase: &parsed.ExpressionBase{
						Location: e.Location,
					},
					Name: ast.QualifiedIdentifier(parts[0]),
				})
				for i := 1; i < len(parts); i++ {
					varAccess = &parsed.Access{
						ExpressionBase: &parsed.ExpressionBase{
							Location: e.Location,
						},
						Record:    varAccess,
						FieldName: ast.Identifier(parts[i]),
					}
				}
				return normalizeExpression(locals, modules, module, varAccess, normalizedModule)
			}

			return nil, common.Error{
				Location: e.Location,
				Message:  fmt.Sprintf("identifier `%s` not found", e.Location.Text()),
			}
		}
	case *parsed.InfixVar:
		{
			e := expr.(*parsed.InfixVar)
			if i, m, ids := findParsedInfixFn(modules, module, e.Infix); len(ids) != 1 {
				return nil, newAmbiguousInfixError(ids, e.Infix, e.Location)
			} else if d, _, ids := FindParsedDefinition(nil, m, ast.QualifiedIdentifier(i.Alias), normalizedModule); len(ids) != 1 {
				return nil, newAmbiguousDefinitionError(ids, ast.QualifiedIdentifier(i.Alias), e.Location)
			} else {
				return e.SetSuccessor(normalized.Global{
					ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
					ModuleName:     m.Name,
					DefinitionName: d.Name,
				}), nil
			}
		}
	}
	return nil, common.NewCompilerError("impossible case")
}

func extractUsedLocals(
	expr normalized.Expression, definedLocals map[ast.Identifier]normalized.Pattern,
	params map[ast.Identifier]normalized.Pattern,
) ([]ast.Identifier, error) {
	usedLocals := map[ast.Identifier]struct{}{}
	if err := extractUsedLocalsSet(expr, definedLocals, usedLocals); err != nil {
		return nil, err
	}
	var uniqueLocals []ast.Identifier
	for k := range usedLocals {
		if _, ok := params[k]; !ok {
			uniqueLocals = append(uniqueLocals, k)
		}
	}
	return uniqueLocals, nil
}

func extractUsedLocalsSet(
	expr normalized.Expression,
	definedLocals map[ast.Identifier]normalized.Pattern,
	usedLocals map[ast.Identifier]struct{},
) error {
	switch expr.(type) {
	case normalized.Local:
		{
			e := expr.(normalized.Local)
			if _, ok := definedLocals[e.Name]; ok {
				usedLocals[e.Name] = struct{}{}
			}
		}
	case normalized.Access:
		{
			e := expr.(normalized.Access)
			if err := extractUsedLocalsSet(e.Record, definedLocals, usedLocals); err != nil {
				return err
			}
			break
		}
	case normalized.Apply:
		{
			e := expr.(normalized.Apply)
			if err := extractUsedLocalsSet(e.Func, definedLocals, usedLocals); err != nil {
				return err
			}
			for _, a := range e.Args {
				if err := extractUsedLocalsSet(a, definedLocals, usedLocals); err != nil {
					return err
				}
			}
			break
		}
	case normalized.Const:
		{
			break
		}
	case normalized.LetMatch:
		{
			e := expr.(normalized.LetMatch)
			if err := extractUsedLocalsSet(e.Value, definedLocals, usedLocals); err != nil {
				return err
			}
			if err := extractUsedLocalsSet(e.Nested, definedLocals, usedLocals); err != nil {
				return err
			}
			break
		}
	case normalized.LetDef:
		{
			e := expr.(normalized.LetDef)
			if err := extractUsedLocalsSet(e.Body, definedLocals, usedLocals); err != nil {
				return err
			}
			if err := extractUsedLocalsSet(e.Nested, definedLocals, usedLocals); err != nil {
				return err
			}
			break
		}
	case normalized.List:
		{
			e := expr.(normalized.List)
			for _, i := range e.Items {
				if err := extractUsedLocalsSet(i, definedLocals, usedLocals); err != nil {
					return err
				}
			}
			break
		}
	case normalized.Record:
		{
			e := expr.(normalized.Record)
			for _, f := range e.Fields {
				if err := extractUsedLocalsSet(f.Value, definedLocals, usedLocals); err != nil {
					return err
				}
			}
			break
		}
	case normalized.Select:
		{
			e := expr.(normalized.Select)
			if err := extractUsedLocalsSet(e.Condition, definedLocals, usedLocals); err != nil {
				return err
			}
			for _, c := range e.Cases {
				if err := extractUsedLocalsSet(c.Expression, definedLocals, usedLocals); err != nil {
					return err
				}
			}
			break
		}
	case normalized.Tuple:
		{
			e := expr.(normalized.Tuple)
			for _, i := range e.Items {
				if err := extractUsedLocalsSet(i, definedLocals, usedLocals); err != nil {
					return err
				}
			}
			break
		}
	case normalized.UpdateLocal:
		{
			e := expr.(normalized.UpdateLocal)
			for _, f := range e.Fields {
				if err := extractUsedLocalsSet(f.Value, definedLocals, usedLocals); err != nil {
					return err
				}
			}
			break
		}
	case normalized.UpdateGlobal:
		{
			e := expr.(normalized.UpdateGlobal)
			for _, f := range e.Fields {
				if err := extractUsedLocalsSet(f.Value, definedLocals, usedLocals); err != nil {
					return err
				}
			}
			break
		}
	case normalized.Constructor:
		{
			e := expr.(normalized.Constructor)
			for _, a := range e.Args {
				if err := extractUsedLocalsSet(a, definedLocals, usedLocals); err != nil {
					return err
				}
			}
			break
		}
	case normalized.NativeCall:
		{
			e := expr.(normalized.NativeCall)
			for _, a := range e.Args {
				if err := extractUsedLocalsSet(a, definedLocals, usedLocals); err != nil {
					return err
				}
			}
			break
		}
	case normalized.Global:
		{
			break
		}
	case normalized.Lambda:
		{
			e := expr.(normalized.Lambda)
			if err := extractUsedLocalsSet(e.Body, definedLocals, usedLocals); err != nil {
				return err
			}
			break
		}
	default:
		return common.NewCompilerError("impossible case")
	}
	return nil
}

func normalizeType(
	modules map[ast.QualifiedIdentifier]*parsed.Module, module *parsed.Module, typeModule *parsed.Module, t parsed.Type,
	namedTypes namedTypeMap,
) (normalized.Type, error) {
	if t == nil {
		return nil, nil
	}
	normalize := func(x parsed.Type) (normalized.Type, error) {
		return normalizeType(modules, module, typeModule, x, namedTypes)
	}
	switch t.(type) {
	case *parsed.TFunc:
		{
			e := t.(*parsed.TFunc)
			params, err := common.MapError(normalize, e.Params)
			if err != nil {
				return nil, err
			}
			ret, err := normalize(e.Return)
			if err != nil {
				return nil, err
			}
			return normalized.Type(&normalized.TFunc{
				Location: e.Location,
				Params:   params,
				Return:   ret,
			}), nil
		}
	case *parsed.TRecord:
		{
			e := t.(*parsed.TRecord)
			fields := map[ast.Identifier]normalized.Type{}
			for n, v := range e.Fields {
				var err error
				fields[n], err = normalize(v)
				if err != nil {
					return nil, err
				}
			}
			return &normalized.TRecord{
				Location: e.Location,
				Fields:   fields,
			}, nil
		}
	case *parsed.TTuple:
		{
			e := t.(*parsed.TTuple)
			items, err := common.MapError(normalize, e.Items)
			if err != nil {
				return nil, err
			}
			return &normalized.TTuple{
				Location: e.Location,
				Items:    items,
			}, nil
		}
	case *parsed.TUnit:
		{
			e := t.(*parsed.TUnit)
			return &normalized.TUnit{
				Location: e.Location,
			}, nil
		}
	case *parsed.TData:
		{
			e := t.(*parsed.TData)
			if namedTypes == nil {
				namedTypes = namedTypeMap{}
			}
			if placeholder, cached := namedTypes[e.Name]; cached {
				return placeholder, nil
			}
			namedTypes[e.Name] = &normalized.TPlaceholder{
				Name: e.Name,
			}

			args, err := common.MapError(normalize, e.Args)
			if err != nil {
				return nil, err
			}
			options, err := common.MapError(func(x parsed.DataOption) (normalized.DataOption, error) {
				values, err := common.MapError(func(x parsed.Type) (normalized.Type, error) {
					if typeModule != nil {
						return normalizeType(modules, typeModule, nil, x, namedTypes)
					} else {
						return normalizeType(modules, module, nil, x, namedTypes)
					}
				}, x.Values)
				if err != nil {
					return normalized.DataOption{}, err
				}
				return normalized.DataOption{
					Name:   x.Name,
					Hidden: x.Hidden,
					Values: values,
				}, nil
			}, e.Options)

			return &normalized.TData{
				Location: e.Location,
				Name:     e.Name,
				Args:     args,
				Options:  options,
			}, nil
		}
	case *parsed.TNative:
		{
			e := t.(*parsed.TNative)
			args, err := common.MapError(normalize, e.Args)
			if err != nil {
				return nil, err
			}
			return &normalized.TNative{
				Location: e.Location,
				Name:     e.Name,
				Args:     args,
			}, nil
		}
	case *parsed.TTypeParameter:
		{
			e := t.(*parsed.TTypeParameter)
			return &normalized.TTypeParameter{
				Location: e.Location,
				Name:     e.Name,
			}, nil
		}
	case *parsed.TNamed:
		{
			e := t.(*parsed.TNamed)
			x, m, ids, err := FindParsedType(modules, module, e.Name, e.Args)
			if ids == nil && typeModule != nil {
				x, m, ids, err = FindParsedType(modules, typeModule, e.Name, e.Args)
				if err != nil {
					return nil, err
				}
			}
			if ids == nil {
				return nil, common.Error{Location: e.Location, Message: fmt.Sprintf("type `%s` not found", e.Name)}
			}
			if len(ids) > 1 {
				return nil, common.Error{
					Location: e.Location,
					Message: fmt.Sprintf(
						"ambiguous type `%s`, it can be one of %s. Use import or qualified name to clarify which one to use",
						e.Name, common.Join(ids, ", ")),
				}
			}
			if named, ok := x.(*parsed.TNamed); ok {
				if named.Name == e.Name {
					return nil, common.Error{
						Location: named.Location,
						Message:  fmt.Sprintf("type `%s` aliased to itself", e.Name),
					}
				}
			}

			return normalizeType(modules, module, m, x, namedTypes)
		}
	}
	return nil, common.NewCompilerError("impossible case")
}

func FindParsedDefinition(
	modules map[ast.QualifiedIdentifier]*parsed.Module,
	module *parsed.Module,
	name ast.QualifiedIdentifier,
	normalizedModule *normalized.Module,
) (*parsed.Definition, *parsed.Module, []ast.FullIdentifier) {
	d, m, id := findParsedDefinitionImpl(modules, module, name)
	if len(id) == 1 {
		normalizedModule.Dependencies[m.Name] =
			append(normalizedModule.Dependencies[m.Name], d.Name)
	}
	return d, m, id
}

func findParsedDefinitionImpl(
	modules map[ast.QualifiedIdentifier]*parsed.Module, module *parsed.Module, name ast.QualifiedIdentifier,
) (*parsed.Definition, *parsed.Module, []ast.FullIdentifier) {
	var defNameEq = func(x *parsed.Definition) bool {
		return ast.QualifiedIdentifier(x.Name) == name
	}

	//1. search in current module
	if def, ok := common.Find(defNameEq, module.Definitions); ok {
		return def, module, []ast.FullIdentifier{common.MakeFullIdentifier(module.Name, def.Name)}
	}

	lastDot := strings.LastIndex(string(name), ".")
	defName := name[lastDot+1:]
	modName := ""
	if lastDot >= 0 {
		modName = string(name[:lastDot])
	}

	//2. search in imported modules
	if modules != nil {
		for _, imp := range module.Imports {
			if slices.Contains(imp.Exposing, string(name)) {
				return findParsedDefinitionImpl(nil, modules[imp.ModuleIdentifier], defName)
			}
		}

		var rDef *parsed.Definition
		var rModule *parsed.Module
		var rIdent []ast.FullIdentifier

		//3. search in all modules by qualified name
		if modName != "" {
			if submodule, ok := modules[ast.QualifiedIdentifier(modName)]; ok {
				if _, referenced := module.ReferencedPackages[submodule.PackageName]; referenced {
					return findParsedDefinitionImpl(nil, submodule, defName)
				}
			}

			//4. search in all modules by short name
			modName = "." + modName
			for modId, submodule := range modules {
				if _, referenced := module.ReferencedPackages[submodule.PackageName]; referenced {
					if strings.HasSuffix(string(modId), modName) {
						if d, m, i := findParsedDefinitionImpl(nil, submodule, defName); len(i) != 0 {
							rDef = d
							rModule = m
							rIdent = append(rIdent, i...)
						}
					}
				}
			}
			if len(rIdent) != 0 {
				return rDef, rModule, rIdent
			}
		}

		//5. search by definition name as module name
		if len(defName) > 0 && unicode.IsUpper([]rune(defName)[0]) {
			modDotName := string("." + defName)
			for modId, submodule := range modules {
				if _, referenced := module.ReferencedPackages[submodule.PackageName]; referenced {
					if strings.HasSuffix(string(modId), modDotName) || modId == defName {
						if d, m, i := findParsedDefinitionImpl(nil, submodule, defName); len(i) != 0 {
							rDef = d
							rModule = m
							rIdent = append(rIdent, i...)
						}
					}
				}
			}
			if len(rIdent) != 0 {
				return rDef, rModule, rIdent
			}
		}

		if modName == "" {
			//6. search all modules
			for _, submodule := range modules {
				if _, referenced := module.ReferencedPackages[submodule.PackageName]; referenced {
					if d, m, i := findParsedDefinitionImpl(nil, submodule, defName); len(i) != 0 {
						rDef = d
						rModule = m
						rIdent = append(rIdent, i...)
					}
				}
			}
			if len(rIdent) != 0 {
				return rDef, rModule, rIdent
			}
		}
	}

	return nil, nil, nil
}

func findParsedInfixFn(
	modules map[ast.QualifiedIdentifier]*parsed.Module, module *parsed.Module, name ast.InfixIdentifier,
) (parsed.Infix, *parsed.Module, []ast.FullIdentifier) {
	//1. search in current module
	var infNameEq = func(x parsed.Infix) bool { return x.Name == name }
	if inf, ok := common.Find(infNameEq, module.InfixFns); ok {
		id := common.MakeFullIdentifier(module.Name, inf.Alias)
		return inf, module, []ast.FullIdentifier{id}
	}

	//2. search in imported modules
	if modules != nil {
		for _, imp := range module.Imports {
			if slices.Contains(imp.Exposing, string(name)) {
				return findParsedInfixFn(nil, modules[imp.ModuleIdentifier], name)
			}
		}

		//6. search all modules
		var rInfix parsed.Infix
		var rModule *parsed.Module
		var rIdent []ast.FullIdentifier
		for _, submodule := range modules {
			if _, referenced := module.ReferencedPackages[submodule.PackageName]; referenced {
				if foundInfix, foundModule, foundId := findParsedInfixFn(nil, submodule, name); foundId != nil {
					rInfix = foundInfix
					rModule = foundModule
					rIdent = append(rIdent, foundId...)
				}
			}
		}
		return rInfix, rModule, rIdent
	}
	return parsed.Infix{}, nil, nil
}

func FindParsedType(
	modules map[ast.QualifiedIdentifier]*parsed.Module,
	module *parsed.Module,
	name ast.QualifiedIdentifier,
	args []parsed.Type,
) (parsed.Type, *parsed.Module, []ast.FullIdentifier, error) {
	var aliasNameEq = func(x parsed.Alias) bool {
		return ast.QualifiedIdentifier(x.Name) == name
	}

	// 1. check current module
	if alias, ok := common.Find(aliasNameEq, module.Aliases); ok {
		id := common.MakeFullIdentifier(module.Name, alias.Name)
		if alias.Type == nil {
			return &parsed.TNative{
				Location: alias.Location,
				Name:     id,
				Args:     args,
			}, module, []ast.FullIdentifier{id}, nil
		}
		if len(alias.Params) != len(args) {
			return nil, nil, nil, nil
		}
		typeMap := map[ast.Identifier]parsed.Type{}
		for i, x := range alias.Params {
			typeMap[x] = args[i]
		}
		withAppliedArgs, err := applyTypeArgs(alias.Type, typeMap)
		if err != nil {
			return nil, nil, nil, err
		}
		return withAppliedArgs, module, []ast.FullIdentifier{id}, nil
	}

	lastDot := strings.LastIndex(string(name), ".")
	typeName := name[lastDot+1:]
	modName := ""
	if lastDot >= 0 {
		modName = string(name[:lastDot])
	}

	//2. search in imported modules
	if modules != nil {
		var rType parsed.Type
		var rModule *parsed.Module
		var rIdent []ast.FullIdentifier

		for _, imp := range module.Imports {
			if slices.Contains(imp.Exposing, string(name)) {
				return FindParsedType(nil, modules[imp.ModuleIdentifier], typeName, args)
			}
		}

		//3. search in all modules by qualified name
		if modName != "" {
			if submodule, ok := modules[ast.QualifiedIdentifier(modName)]; ok {
				if _, referenced := module.ReferencedPackages[submodule.PackageName]; referenced {
					return FindParsedType(nil, submodule, typeName, args)
				}
			}

			//4. search in all modules by short name
			modName = "." + modName
			for modId, submodule := range modules {
				if _, referenced := module.ReferencedPackages[submodule.PackageName]; referenced {
					if strings.HasSuffix(string(modId), modName) {
						foundType, foundModule, foundId, err := FindParsedType(nil, submodule, typeName, args)
						if err != nil {
							return nil, nil, nil, err
						}
						if foundId != nil {
							rType = foundType
							rModule = foundModule
							rIdent = append(rIdent, foundId...)
						}
					}
				}
			}
			if len(rIdent) != 0 {
				return rType, rModule, rIdent, nil
			}
		}

		//5. search by type name as module name
		if unicode.IsUpper([]rune(typeName)[0]) {
			modDotName := string("." + typeName)
			for modId, submodule := range modules {
				if _, referenced := module.ReferencedPackages[submodule.PackageName]; referenced {
					if strings.HasSuffix(string(modId), modDotName) || modId == typeName {
						foundType, foundModule, foundId, err := FindParsedType(nil, submodule, typeName, args)
						if err != nil {
							return nil, nil, nil, err
						}
						if foundId != nil {
							rType = foundType
							rModule = foundModule
							rIdent = append(rIdent, foundId...)
						}
					}
				}
			}
			if len(rIdent) != 0 {
				return rType, rModule, rIdent, nil
			}
		}

		if modName == "" {
			//6. search all modules
			for _, submodule := range modules {
				if _, referenced := module.ReferencedPackages[submodule.PackageName]; referenced {
					foundType, foundModule, foundId, err := FindParsedType(nil, submodule, typeName, args)
					if err != nil {
						return nil, nil, nil, err
					}
					if foundId != nil {
						rType = foundType
						rModule = foundModule
						rIdent = append(rIdent, foundId...)
					}
				}
			}
			if len(rIdent) != 0 {
				return rType, rModule, rIdent, nil
			}
		}
	}

	return nil, nil, nil, nil
}

func applyTypeArgs(t parsed.Type, params map[ast.Identifier]parsed.Type) (parsed.Type, error) {
	doMap := func(x parsed.Type) (parsed.Type, error) { return applyTypeArgs(x, params) }
	var err error
	switch t.(type) {
	case *parsed.TFunc:
		{
			p := t.(*parsed.TFunc)
			e := &parsed.TFunc{
				Location: p.Location,
				Params:   nil,
				Return:   nil,
			}
			e.Params, err = common.MapError(doMap, p.Params)
			if err != nil {
				return nil, err
			}
			e.Return, err = applyTypeArgs(p.Return, params)
			if err != nil {
				return nil, err
			}
			return e, nil
		}
	case *parsed.TRecord:
		{
			p := t.(*parsed.TRecord)
			e := &parsed.TRecord{
				Location: p.Location,
				Fields:   map[ast.Identifier]parsed.Type{},
			}
			for name, f := range p.Fields {
				e.Fields[name], err = applyTypeArgs(f, params)
				if err != nil {
					return nil, err
				}
			}
			return e, nil
		}
	case *parsed.TTuple:
		{
			p := t.(*parsed.TTuple)
			e := &parsed.TTuple{
				Location: p.Location,
				Items:    nil,
			}
			e.Items, err = common.MapError(doMap, p.Items)
			if err != nil {
				return nil, err
			}
			return e, nil
		}
	case *parsed.TUnit:
		return t, nil
	case *parsed.TData:
		{
			p := t.(*parsed.TData)
			e := &parsed.TData{
				Location: p.Location,
				Name:     p.Name,
				Options:  p.Options,
				Args:     nil,
			}
			e.Args, err = common.MapError(doMap, p.Args)
			if err != nil {
				return nil, err
			}
			return e, nil
		}
	case *parsed.TNamed:
		{
			p := t.(*parsed.TNamed)
			e := &parsed.TNamed{
				Location: p.Location,
				Name:     p.Name,
				Args:     nil,
			}
			e.Args, err = common.MapError(doMap, p.Args)
			if err != nil {
				return nil, err
			}
			return e, nil
		}
	case *parsed.TNative:
		{
			p := t.(*parsed.TNative)
			e := &parsed.TNative{
				Location: p.Location,
				Name:     p.Name,
				Args:     nil,
			}
			e.Args, err = common.MapError(doMap, p.Args)
			if err != nil {
				return nil, err
			}
			return e, nil
		}
	case *parsed.TTypeParameter:
		{
			e := t.(*parsed.TTypeParameter)
			return params[e.Name], nil
		}
	}
	return nil, common.NewCompilerError("impossible case")
}

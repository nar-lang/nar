package processors

import (
	"fmt"
	"maps"
	"oak-compiler/internal/pkg/ast"
	"oak-compiler/internal/pkg/ast/normalized"
	"oak-compiler/internal/pkg/ast/parsed"
	"oak-compiler/internal/pkg/common"
	"slices"
	"strings"
)

var lastDefinitionId = uint64(0)
var lastLambdaId = uint64(0)

func Normalize(
	moduleName ast.QualifiedIdentifier,
	modules map[ast.QualifiedIdentifier]*parsed.Module,
	normalizedModules map[ast.QualifiedIdentifier]*normalized.Module,
) bool {
	if _, ok := normalizedModules[moduleName]; ok {
		return true
	}

	m, ok := modules[moduleName]
	if !ok {
		return false
	}

	for _, imp := range m.Imports {
		if !Normalize(imp.ModuleIdentifier, modules, normalizedModules) {
			panic(common.Error{Location: imp.Location, Message: fmt.Sprintf("module `%s` not found", imp.ModuleIdentifier)})
		}
	}

	flattenDataTypes(m)
	unwrapImports(m, modules)
	modules[moduleName] = m

	o := &normalized.Module{
		Name: m.Name,
	}

	lastLambdaId = 0

	for _, def := range m.Definitions {
		nDef, params := normalizeDefinition(modules, m, def)
		nDef.Expression = flattenLambdas(nDef.Expression, o, params)
		o.Definitions = append(o.Definitions, &nDef)
	}

	for _, imp := range m.Imports {
		o.Dependencies = append(o.Dependencies, imp.ModuleIdentifier)
	}

	normalizedModules[m.Name] = o
	return true
}

func extractLocals(pattern normalized.Pattern, locals map[ast.Identifier]struct{}) {
	switch pattern.(type) {
	case normalized.PAlias:
		{
			e := pattern.(normalized.PAlias)
			locals[e.Alias] = struct{}{}
			extractLocals(e.Nested, locals)
			break
		}
	case normalized.PAny:
		{
			break
		}
	case normalized.PCons:
		{
			e := pattern.(normalized.PCons)
			extractLocals(e.Head, locals)
			extractLocals(e.Tail, locals)
			break
		}
	case normalized.PConst:
		{
			break
		}
	case normalized.PDataOption:
		{
			e := pattern.(normalized.PDataOption)
			for _, v := range e.Values {
				extractLocals(v, locals)
			}
			break
		}
	case normalized.PList:
		{
			e := pattern.(normalized.PList)
			for _, v := range e.Items {
				extractLocals(v, locals)
			}
			break
		}
	case normalized.PNamed:
		{
			e := pattern.(normalized.PNamed)
			locals[e.Name] = struct{}{}
			break
		}
	case normalized.PRecord:
		{
			e := pattern.(normalized.PRecord)
			for _, v := range e.Fields {
				locals[v.Name] = struct{}{}
			}
			break
		}
	case normalized.PTuple:
		{
			e := pattern.(normalized.PTuple)
			for _, v := range e.Items {
				extractLocals(v, locals)
			}
			break
		}
	default:
		panic(common.SystemError{Message: "impossible case"})
	}
}

func extractLambda(
	loc ast.Location, params []normalized.Pattern, body normalized.Expression,
	m *normalized.Module, locals map[ast.Identifier]struct{},
) (def *normalized.Definition, usedLocals []ast.Identifier, replacement normalized.Expression) {
	lastLambdaId++
	lambdaName := ast.Identifier(fmt.Sprintf("_lambda_%d", lastLambdaId))
	usedLocals = extractUsedLocals(body, locals)

	lastDefinitionId++
	def = &normalized.Definition{
		Id:   lastDefinitionId,
		Name: lambdaName,
		Params: append(
			common.Map(func(x ast.Identifier) normalized.Pattern {
				return normalized.PNamed{Location: loc, Name: x}
			}, usedLocals),
			params...),
		Expression: body,
		Location:   loc,
		Hidden:     true,
	}
	m.Definitions = append(m.Definitions, def)

	replacement = normalized.Global{
		Location:       loc,
		ModuleName:     m.Name,
		DefinitionName: def.Name,
	}

	if len(usedLocals) > 0 {
		replacement = normalized.Apply{
			Location: loc,
			Func:     replacement,
			Args: common.Map(func(x ast.Identifier) normalized.Expression {
				return normalized.Local{
					Location: loc,
					Name:     x,
				}
			}, usedLocals),
		}
	}

	return
}

func extractParameters(def *normalized.Definition) map[ast.Identifier]struct{} {
	params := map[ast.Identifier]struct{}{}
	for _, p := range def.Params {
		extractLocals(p, params)
	}
	return params
}

func flattenLambdas(
	expr normalized.Expression, m *normalized.Module, locals map[ast.Identifier]struct{},
) normalized.Expression {
	switch expr.(type) {
	case normalized.Lambda:
		{
			e := expr.(normalized.Lambda)
			def, _, replacement := extractLambda(e.Location, e.Params, e.Body, m, locals)
			def.Expression = flattenLambdas(def.Expression, m, extractParameters(def))
			return replacement
		}
	case normalized.LetDef:
		{
			e := expr.(normalized.LetDef)
			def, usedLocals, replacement := extractLambda(e.Location, e.Params, e.Body, m, locals)

			if len(usedLocals) > 0 {
				replName := ast.Identifier(fmt.Sprintf("_lambda_closue_%d", lastLambdaId))
				replaceMap := map[ast.Identifier]normalized.Expression{}
				replaceMap[e.Name] = normalized.Local{
					Location: e.Location,
					Name:     replName,
				}

				let := normalized.LetMatch{
					Location: e.Location,
					Pattern: normalized.PNamed{
						Location: e.Location,
						Name:     replName,
					},
					Value:  replacement,
					Nested: def.Expression,
				}
				def.Expression = let
				def.Expression = replaceLocals(def.Expression, replaceMap)
				def.Expression = flattenLambdas(def.Expression, m, extractParameters(def))

				let.Nested = replaceLocals(e.Nested, replaceMap)
				let.Nested = flattenLambdas(let.Nested, m, locals)
				return let
			} else {
				replaceMap := map[ast.Identifier]normalized.Expression{}
				replaceMap[e.Name] = replacement

				def.Expression = replaceLocals(def.Expression, replaceMap)
				def.Expression = flattenLambdas(def.Expression, m, extractParameters(def))

				return flattenLambdas(replaceLocals(e.Nested, replaceMap), m, locals)
			}
		}
	case normalized.LetMatch:
		{
			e := expr.(normalized.LetMatch)
			innerLocals := maps.Clone(locals)
			extractLocals(e.Pattern, innerLocals)
			e.Value = flattenLambdas(e.Value, m, innerLocals)
			e.Nested = flattenLambdas(e.Nested, m, innerLocals)
			return e
		}
	case normalized.Access:
		{
			e := expr.(normalized.Access)
			e.Record = flattenLambdas(e.Record, m, locals)
			return e
		}
	case normalized.Apply:
		{
			e := expr.(normalized.Apply)
			e.Func = flattenLambdas(e.Func, m, locals)
			for i, a := range e.Args {
				e.Args[i] = flattenLambdas(a, m, locals)
			}
			return e
		}
	case normalized.If:
		{
			e := expr.(normalized.If)
			e.Condition = flattenLambdas(e.Condition, m, locals)
			e.Positive = flattenLambdas(e.Positive, m, locals)
			e.Negative = flattenLambdas(e.Negative, m, locals)
			return e
		}
	case normalized.List:
		{
			e := expr.(normalized.List)
			for i, a := range e.Items {
				e.Items[i] = flattenLambdas(a, m, locals)
			}
			return e
		}
	case normalized.Record:
		{
			e := expr.(normalized.Record)
			for i, a := range e.Fields {
				e.Fields[i].Value = flattenLambdas(a.Value, m, locals)
			}
			return e
		}
	case normalized.Select:
		{
			e := expr.(normalized.Select)
			e.Condition = flattenLambdas(e.Condition, m, locals)
			for i, a := range e.Cases {
				innerLocals := maps.Clone(locals)
				extractLocals(a.Pattern, innerLocals)
				e.Cases[i].Expression = flattenLambdas(a.Expression, m, innerLocals)
			}
			return e
		}
	case normalized.Tuple:
		{
			e := expr.(normalized.Tuple)
			for i, a := range e.Items {
				e.Items[i] = flattenLambdas(a, m, locals)
			}
			return e
		}
	case normalized.UpdateLocal:
		{
			e := expr.(normalized.UpdateLocal)
			for i, a := range e.Fields {
				e.Fields[i].Value = flattenLambdas(a.Value, m, locals)
			}
			return e
		}
	case normalized.UpdateGlobal:
		{
			e := expr.(normalized.UpdateGlobal)
			for i, a := range e.Fields {
				e.Fields[i].Value = flattenLambdas(a.Value, m, locals)
			}
			return e
		}
	case normalized.Constructor:
		{
			e := expr.(normalized.Constructor)
			for i, a := range e.Args {
				e.Args[i] = flattenLambdas(a, m, locals)
			}
			return e
		}
	case normalized.NativeCall:
		{
			e := expr.(normalized.NativeCall)
			for i, a := range e.Args {
				e.Args[i] = flattenLambdas(a, m, locals)
			}
			return e
		}
	case normalized.Const:
		{
			return expr
		}
	case normalized.Global:
		{
			return expr
		}
	case normalized.Local:
		{
			return expr
		}
	default:
		panic(common.SystemError{Message: "invalid case"})
	}
}

func replaceLocals(expr normalized.Expression, replace map[ast.Identifier]normalized.Expression) normalized.Expression {
	switch expr.(type) {
	case normalized.Lambda:
		{
			e := expr.(normalized.Lambda)
			e.Body = replaceLocals(e.Body, replace)
			return e
		}
	case normalized.LetDef:
		{
			e := expr.(normalized.LetDef)
			e.Body = replaceLocals(e.Body, replace)
			e.Nested = replaceLocals(e.Nested, replace)
			return e
		}
	case normalized.LetMatch:
		{
			e := expr.(normalized.LetMatch)
			e.Value = replaceLocals(e.Value, replace)
			e.Nested = replaceLocals(e.Nested, replace)
			return e
		}
	case normalized.Access:
		{
			e := expr.(normalized.Access)
			e.Record = replaceLocals(e.Record, replace)
			return e
		}
	case normalized.Apply:
		{
			e := expr.(normalized.Apply)
			e.Func = replaceLocals(e.Func, replace)
			for i, a := range e.Args {
				e.Args[i] = replaceLocals(a, replace)
			}
			return e
		}
	case normalized.If:
		{
			e := expr.(normalized.If)
			e.Condition = replaceLocals(e.Condition, replace)
			e.Positive = replaceLocals(e.Positive, replace)
			e.Negative = replaceLocals(e.Negative, replace)
			return e
		}
	case normalized.List:
		{
			e := expr.(normalized.List)
			for i, a := range e.Items {
				e.Items[i] = replaceLocals(a, replace)
			}
			return e
		}
	case normalized.Record:
		{
			e := expr.(normalized.Record)
			for i, a := range e.Fields {
				e.Fields[i].Value = replaceLocals(a.Value, replace)
			}
			return e
		}
	case normalized.Select:
		{
			e := expr.(normalized.Select)
			e.Condition = replaceLocals(e.Condition, replace)
			for i, a := range e.Cases {
				e.Cases[i].Expression = replaceLocals(a.Expression, replace)
			}
			return e
		}
	case normalized.Tuple:
		{
			e := expr.(normalized.Tuple)
			for i, a := range e.Items {
				e.Items[i] = replaceLocals(a, replace)
			}
			return e
		}
	case normalized.UpdateLocal:
		{
			e := expr.(normalized.UpdateLocal)
			for i, a := range e.Fields {
				e.Fields[i].Value = replaceLocals(a.Value, replace)
			}
			return e
		}
	case normalized.UpdateGlobal:
		{
			e := expr.(normalized.UpdateGlobal)
			for i, a := range e.Fields {
				e.Fields[i].Value = replaceLocals(a.Value, replace)
			}
			return e
		}
	case normalized.Constructor:
		{
			e := expr.(normalized.Constructor)
			for i, a := range e.Args {
				e.Args[i] = replaceLocals(a, replace)
			}
			return e
		}
	case normalized.NativeCall:
		{
			e := expr.(normalized.NativeCall)
			for i, a := range e.Args {
				e.Args[i] = replaceLocals(a, replace)
			}
			return e
		}
	case normalized.Const:
		{
			return expr
		}
	case normalized.Global:
		{
			return expr
		}
	case normalized.Local:
		{
			e := expr.(normalized.Local)
			if r, ok := replace[e.Name]; ok {
				return r
			}
			return expr
		}
	default:
		panic(common.SystemError{Message: "invalid case"})
	}
}

func flattenDataTypes(m *parsed.Module) {
	for _, it := range m.DataTypes {
		typeArgs := common.Map(func(x ast.Identifier) parsed.Type {
			return parsed.TTypeParameter{
				Location: it.Location,
				Name:     x,
			}
		}, it.Params)
		m.Aliases = append(m.Aliases, parsed.Alias{
			Location: it.Location,
			Name:     it.Name,
			Params:   it.Params,
			Type: parsed.TData{
				Location: it.Location,
				Name:     common.MakeExternalIdentifier(m.Name, it.Name),
				Args:     typeArgs,
				Options:  common.Map(func(x parsed.DataTypeOption) ast.Identifier { return x.Name }, it.Options),
			},
		})
		for _, option := range it.Options {
			var type_ parsed.Type = parsed.TExternal{
				Location: it.Location,
				Name:     common.MakeExternalIdentifier(m.Name, it.Name),
				Args:     typeArgs,
			}
			if len(option.Params) > 0 {
				type_ = parsed.TFunc{
					Location: it.Location,
					Params:   option.Params,
					Return:   type_,
				}
			}
			var body parsed.Expression = parsed.Constructor{
				Location:   option.Location,
				DataName:   common.MakeExternalIdentifier(m.Name, it.Name),
				OptionName: option.Name,
				Args: common.Map(
					func(i int) parsed.Expression {
						return parsed.Var{
							Location: option.Location,
							Name:     ast.QualifiedIdentifier(fmt.Sprintf("p%d", i)),
						}
					},
					common.Range(0, len(option.Params)),
				),
			}

			params := common.Map(
				func(i int) parsed.Pattern {
					return parsed.PNamed{Location: option.Location, Name: ast.Identifier(fmt.Sprintf("p%d", i))}
				},
				common.Range(0, len(option.Params)),
			)

			m.Definitions = append(m.Definitions, parsed.Definition{
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

func unwrapImports(module *parsed.Module, modules map[ast.QualifiedIdentifier]*parsed.Module) {
	for i, imp := range module.Imports {
		m := modules[imp.ModuleIdentifier]
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

		for _, d := range m.Definitions {
			n := string(d.Name)
			if imp.ExposingAll || slices.Contains(imp.Exposing, n) {
				exp = append(exp, n)
			}
			exp = append(exp, fmt.Sprintf("%s.%s", modName, n))
			if shortModName != "" {
				exp = append(exp, fmt.Sprintf("%s.%s", shortModName, n))
			}
		}

		for _, a := range m.Aliases {
			n := string(a.Name)
			if imp.ExposingAll || slices.Contains(imp.Exposing, n) {
				exp = append(exp, n)
				if dt, ok := a.Type.(parsed.TData); ok {
					for _, v := range dt.Options {
						exp = append(exp, string(v))
					}
				}
			}
			exp = append(exp, fmt.Sprintf("%s.%s", modName, n))
			if dt, ok := a.Type.(parsed.TData); ok {
				for _, v := range dt.Options {
					exp = append(exp, fmt.Sprintf("%s.%s", modName, v))
					if shortModName != "" {
						exp = append(exp, fmt.Sprintf("%s.%s", shortModName, v))
					}
				}
			}
		}

		for _, a := range m.InfixFns {
			n := string(a.Name)
			if imp.ExposingAll || slices.Contains(imp.Exposing, n) {
				exp = append(exp, n)
			}
			exp = append(exp, fmt.Sprintf("%s.%s", modName, n))
			if shortModName != "" {
				exp = append(exp, fmt.Sprintf("%s.%s", shortModName, n))
			}
		}
		imp.Exposing = exp
		module.Imports[i] = imp
	}
}

func normalizeDefinition(
	modules map[ast.QualifiedIdentifier]*parsed.Module, module *parsed.Module, def parsed.Definition,
) (normalized.Definition, map[ast.Identifier]struct{}) {
	lastDefinitionId++
	o := normalized.Definition{
		Id:       lastDefinitionId,
		Name:     def.Name,
		Location: def.Location,
		Hidden:   def.Hidden,
	}
	params := map[ast.Identifier]struct{}{}
	o.Params = common.Map(func(x parsed.Pattern) normalized.Pattern {
		return normalizePattern(params, modules, module, x)
	}, def.Params)
	locals := maps.Clone(params)
	o.Expression = normalizeExpression(locals, modules, module, def.Expression)
	o.Type = normalizeType(modules, module, def.Type)
	return o, params
}

func normalizePattern(
	locals map[ast.Identifier]struct{},
	modules map[ast.QualifiedIdentifier]*parsed.Module, module *parsed.Module,
	pattern parsed.Pattern,
) normalized.Pattern {
	normalize := func(p parsed.Pattern) normalized.Pattern { return normalizePattern(locals, modules, module, p) }

	switch pattern.(type) {
	case parsed.PAlias:
		{
			e := pattern.(parsed.PAlias)
			locals[e.Alias] = struct{}{}
			return normalized.PAlias{
				Location: e.Location,
				Type:     normalizeType(modules, module, e.Type),
				Alias:    e.Alias,
				Nested:   normalize(e.Nested),
			}
		}
	case parsed.PAny:
		{
			e := pattern.(parsed.PAny)
			return normalized.PAny{
				Location: e.Location,
				Type:     normalizeType(modules, module, e.Type),
			}
		}
	case parsed.PCons:
		{
			e := pattern.(parsed.PCons)
			return normalized.PCons{
				Location: e.Location,
				Type:     normalizeType(modules, module, e.Type),
				Head:     normalize(e.Head),
				Tail:     normalize(e.Tail),
			}
		}
	case parsed.PConst:
		{
			e := pattern.(parsed.PConst)
			return normalized.PConst{
				Location: e.Location,
				Type:     normalizeType(modules, module, e.Type),
				Value:    e.Value,
			}
		}
	case parsed.PDataOption:
		{
			e := pattern.(parsed.PDataOption)
			mod, def, ok := findParsedDefinition(modules, module, e.Name)
			if !ok {
				panic(common.Error{Location: e.Location, Message: "data constructor not found"})
			}
			return normalized.PDataOption{
				Location:       e.Location,
				Type:           normalizeType(modules, module, e.Type),
				ModuleName:     mod.Name,
				DefinitionName: def.Name,
				Values:         common.Map(normalize, e.Values),
			}
		}
	case parsed.PList:
		{
			e := pattern.(parsed.PList)
			return normalized.PList{
				Location: e.Location,
				Type:     normalizeType(modules, module, e.Type),
				Items:    common.Map(normalize, e.Items),
			}
		}
	case parsed.PNamed:
		{
			e := pattern.(parsed.PNamed)
			locals[e.Name] = struct{}{}
			return normalized.PNamed{
				Location: e.Location,
				Type:     normalizeType(modules, module, e.Type),
				Name:     e.Name,
			}
		}
	case parsed.PRecord:
		{
			e := pattern.(parsed.PRecord)
			return normalized.PRecord{
				Location: e.Location,
				Type:     normalizeType(modules, module, e.Type),
				Fields: common.Map(func(x parsed.PRecordField) normalized.PRecordField {
					return normalized.PRecordField{Location: x.Location, Name: x.Name}
				}, e.Fields),
			}
		}
	case parsed.PTuple:
		{
			e := pattern.(parsed.PTuple)
			return normalized.PTuple{
				Location: e.Location,
				Type:     normalizeType(modules, module, e.Type),
				Items:    common.Map(normalize, e.Items),
			}
		}
	}
	panic(common.SystemError{Message: "impossible case"})
}

func normalizeExpression(
	locals map[ast.Identifier]struct{},
	modules map[ast.QualifiedIdentifier]*parsed.Module, module *parsed.Module,
	expr parsed.Expression,
) normalized.Expression {
	normalize := func(e parsed.Expression) normalized.Expression {
		return normalizeExpression(locals, modules, module, e)
	}
	switch expr.(type) {
	case parsed.Access:
		{
			e := expr.(parsed.Access)
			return normalized.Access{
				Location:  e.Location,
				Record:    normalize(e.Record),
				FieldName: e.FieldName,
			}
		}
	case parsed.Apply:
		{
			e := expr.(parsed.Apply)
			return normalized.Apply{
				Location: e.Location,
				Func:     normalize(e.Func),
				Args:     common.Map(normalize, e.Args),
			}
		}
	case parsed.Const:
		{
			e := expr.(parsed.Const)
			return normalized.Const{
				Location: e.Location,
				Value:    e.Value,
			}
		}
	case parsed.Constructor:
		{
			e := expr.(parsed.Constructor)
			return normalized.Constructor{
				Location:   e.Location,
				DataName:   e.DataName,
				OptionName: e.OptionName,
				Args:       common.Map(normalize, e.Args),
			}
		}
	case parsed.If:
		{
			e := expr.(parsed.If)
			return normalized.If{
				Location:  e.Location,
				Condition: normalize(e.Condition),
				Positive:  normalize(e.Positive),
				Negative:  normalize(e.Negative),
			}
		}
	case parsed.LetMatch:
		{
			e := expr.(parsed.LetMatch)
			innerLocals := maps.Clone(locals)
			return normalized.LetMatch{
				Location: e.Location,
				Pattern:  normalizePattern(innerLocals, modules, module, e.Pattern),
				Value:    normalizeExpression(innerLocals, modules, module, e.Value),
				Nested:   normalizeExpression(innerLocals, modules, module, e.Nested),
			}
		}
	case parsed.LetDef:
		{
			e := expr.(parsed.LetDef)
			innerLocals := maps.Clone(locals)
			innerLocals[e.Name] = struct{}{}
			return normalized.LetDef{
				Location: e.Location,
				Name:     e.Name,
				Params: common.Map(func(x parsed.Pattern) normalized.Pattern {
					return normalizePattern(innerLocals, modules, module, x)
				}, e.Params),
				FnType: normalizeType(modules, module, e.FnType),
				Body:   normalizeExpression(innerLocals, modules, module, e.Body),
				Nested: normalizeExpression(innerLocals, modules, module, e.Nested),
			}
		}
	case parsed.List:
		{
			e := expr.(parsed.List)
			return normalized.List{
				Location: e.Location,
				Items:    common.Map(normalize, e.Items),
			}
		}
	case parsed.NativeCall:
		{
			e := expr.(parsed.NativeCall)
			return normalized.NativeCall{
				Location: e.Location,
				Name:     e.Name,
				Args:     common.Map(normalize, e.Args),
			}
		}
	case parsed.Record:
		{
			e := expr.(parsed.Record)
			return normalized.Record{
				Location: e.Location,
				Fields: common.Map(func(i parsed.RecordField) normalized.RecordField {
					return normalized.RecordField{
						Location: i.Location,
						Name:     i.Name,
						Value:    normalize(i.Value),
					}
				}, e.Fields),
			}
		}
	case parsed.Select:
		{
			//TODO: Check if all cases are exhausting condition
			e := expr.(parsed.Select)
			return normalized.Select{
				Location:  e.Location,
				Condition: normalize(e.Condition),
				Cases: common.Map(func(i parsed.SelectCase) normalized.SelectCase {
					innerLocals := maps.Clone(locals)
					return normalized.SelectCase{
						Location:   e.Location,
						Pattern:    normalizePattern(innerLocals, modules, module, i.Pattern),
						Expression: normalizeExpression(innerLocals, modules, module, i.Expression),
					}
				}, e.Cases),
			}
		}
	case parsed.Tuple:
		{
			e := expr.(parsed.Tuple)
			return normalized.Tuple{
				Location: e.Location,
				Items:    common.Map(normalize, e.Items),
			}
		}
	case parsed.Update:
		{
			e := expr.(parsed.Update)
			if m, d, ok := findParsedDefinition(modules, module, e.RecordName); ok {
				return normalized.UpdateGlobal{
					Location:       e.Location,
					ModuleName:     m.Name,
					DefinitionName: d.Name,
					Fields: common.Map(func(i parsed.RecordField) normalized.RecordField {
						return normalized.RecordField{
							Location: i.Location,
							Name:     i.Name,
							Value:    normalize(i.Value),
						}
					}, e.Fields),
				}
			}

			return normalized.UpdateLocal{
				Location:   e.Location,
				RecordName: ast.Identifier(e.RecordName),
				Fields: common.Map(func(i parsed.RecordField) normalized.RecordField {
					return normalized.RecordField{
						Location: i.Location,
						Name:     i.Name,
						Value:    normalize(i.Value),
					}
				}, e.Fields),
			}
		}
	case parsed.Lambda:
		{
			e := expr.(parsed.Lambda)
			return normalized.Lambda{
				Location: e.Location,
				Params: common.Map(func(x parsed.Pattern) normalized.Pattern {
					return normalizePattern(locals, modules, module, x)
				}, e.Params),
				Body: normalize(e.Body),
			}
		}
	case parsed.Accessor:
		{
			e := expr.(parsed.Accessor)
			return normalize(parsed.Lambda{
				Params: []parsed.Pattern{parsed.PNamed{Location: e.Location, Name: "x"}},
				Body: parsed.Access{
					Location: e.Location,
					Record: parsed.Var{
						Location: e.Location,
						Name:     "x",
					},
					FieldName: e.FieldName,
				},
			})
		}
	case parsed.BinOp:
		{
			e := expr.(parsed.BinOp)
			var output []parsed.BinOpItem
			var operators []parsed.BinOpItem
			for _, o1 := range e.Items {
				if o1.Expression != nil {
					output = append(output, o1)
				} else {
					if _, infixFn, ok := findInfixFn(modules, module, o1.Infix); !ok {
						panic(common.Error{Location: e.Location, Message: "infix op not found"})
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

			var buildTree func() normalized.Expression
			buildTree = func() normalized.Expression {
				op := output[len(output)-1].Infix
				output = output[:len(output)-1]

				if m, infixA, ok := findInfixFn(modules, module, op); !ok {
					panic(common.Error{Location: e.Location, Message: "infix op not found"})
				} else {
					var left, right normalized.Expression
					r := output[len(output)-1]
					if r.Expression != nil {
						right = normalize(r.Expression)
						output = output[:len(output)-1]
					} else {
						right = buildTree()
					}

					l := output[len(output)-1]
					if l.Expression != nil {
						left = normalize(l.Expression)
						output = output[:len(output)-1]
					} else {
						left = buildTree()
					}

					return normalized.Apply{
						Location: e.Location,
						Func: normalized.Global{
							Location:       e.Location,
							ModuleName:     m.Name,
							DefinitionName: infixA.Alias,
						},
						Args: []normalized.Expression{left, right},
					}
				}
			}

			return buildTree()
		}
	case parsed.Negate:
		{
			e := expr.(parsed.Negate)
			return normalized.NativeCall{
				Location: e.Location,
				Name:     common.OakCoreBasicsNeg,
				Args:     []normalized.Expression{normalize(e.Nested)},
			}
		}
	case parsed.Var:
		{
			e := expr.(parsed.Var)
			if _, ok := locals[ast.Identifier(e.Name)]; ok {
				return normalized.Local{
					Location: e.Location,
					Name:     ast.Identifier(e.Name),
				}
			}

			if m, d, ok := findParsedDefinition(modules, module, e.Name); ok {
				return normalized.Global{
					Location:       e.Location,
					ModuleName:     m.Name,
					DefinitionName: d.Name,
				}
			}

			panic(common.Error{Location: e.Location, Message: fmt.Sprintf("identifier `%s` no found", e.Name)})
		}
	case parsed.InfixVar:
		{
			e := expr.(parsed.InfixVar)
			if m, i, ok := findInfixFn(modules, module, e.Infix); !ok {
				panic(common.Error{
					Location: i.AliasLocation,
					Message:  "infix definition not found",
				})
			} else if _, d, ok := findParsedDefinition(nil, m, ast.QualifiedIdentifier(i.Alias)); !ok {
				panic(common.Error{
					Location: i.Location,
					Message:  "infix alias not found",
				})
			} else {
				return normalized.Global{
					Location:       e.Location,
					ModuleName:     m.Name,
					DefinitionName: d.Name,
				}
			}
		}
	}
	panic(common.SystemError{Message: "impossible case"})
}

func extractUsedLocals(
	expr normalized.Expression, definedLocals map[ast.Identifier]struct{},
) []ast.Identifier {
	usedLocals := map[ast.Identifier]struct{}{}
	extractUsedLocalsSet(expr, definedLocals, usedLocals)
	var uniqueLocals []ast.Identifier
	for k := range usedLocals {
		uniqueLocals = append(uniqueLocals, k)
	}
	return uniqueLocals
}

func extractUsedLocalsSet(
	expr normalized.Expression, definedLocals map[ast.Identifier]struct{}, usedLocals map[ast.Identifier]struct{},
) {
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
			extractUsedLocalsSet(e.Record, definedLocals, usedLocals)
			break
		}
	case normalized.Apply:
		{
			e := expr.(normalized.Apply)
			extractUsedLocalsSet(e.Func, definedLocals, usedLocals)
			for _, a := range e.Args {
				extractUsedLocalsSet(a, definedLocals, usedLocals)
			}
			break
		}
	case normalized.Const:
		{
			break
		}
	case normalized.If:
		{
			e := expr.(normalized.If)
			extractUsedLocalsSet(e.Condition, definedLocals, usedLocals)
			extractUsedLocalsSet(e.Positive, definedLocals, usedLocals)
			extractUsedLocalsSet(e.Negative, definedLocals, usedLocals)
			break
		}
	case normalized.LetMatch:
		{
			e := expr.(normalized.LetMatch)
			extractUsedLocalsSet(e.Value, definedLocals, usedLocals)
			extractUsedLocalsSet(e.Nested, definedLocals, usedLocals)
			break
		}
	case normalized.LetDef:
		{
			e := expr.(normalized.LetDef)
			extractUsedLocalsSet(e.Body, definedLocals, usedLocals)
			extractUsedLocalsSet(e.Nested, definedLocals, usedLocals)
			break
		}
	case normalized.List:
		{
			e := expr.(normalized.List)
			for _, i := range e.Items {
				extractUsedLocalsSet(i, definedLocals, usedLocals)
			}
			break
		}
	case normalized.Record:
		{
			e := expr.(normalized.Record)
			for _, f := range e.Fields {
				extractUsedLocalsSet(f.Value, definedLocals, usedLocals)
			}
			break
		}
	case normalized.Select:
		{
			e := expr.(normalized.Select)
			extractUsedLocalsSet(e.Condition, definedLocals, usedLocals)
			for _, c := range e.Cases {
				extractUsedLocalsSet(c.Expression, definedLocals, usedLocals)
			}
			break
		}
	case normalized.Tuple:
		{
			e := expr.(normalized.Tuple)
			for _, i := range e.Items {
				extractUsedLocalsSet(i, definedLocals, usedLocals)
			}
			break
		}
	case normalized.UpdateLocal:
		{
			e := expr.(normalized.UpdateLocal)
			for _, f := range e.Fields {
				extractUsedLocalsSet(f.Value, definedLocals, usedLocals)
			}
			break
		}
	case normalized.UpdateGlobal:
		{
			e := expr.(normalized.UpdateGlobal)
			for _, f := range e.Fields {
				extractUsedLocalsSet(f.Value, definedLocals, usedLocals)
			}
			break
		}
	case normalized.Constructor:
		{
			e := expr.(normalized.Constructor)
			for _, a := range e.Args {
				extractUsedLocalsSet(a, definedLocals, usedLocals)
			}
			break
		}
	case normalized.NativeCall:
		{
			e := expr.(normalized.NativeCall)
			for _, a := range e.Args {
				extractUsedLocalsSet(a, definedLocals, usedLocals)
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
			extractUsedLocalsSet(e.Body, definedLocals, usedLocals)
		}
	default:
		panic(common.SystemError{Message: "invalid case"})
	}
}

func normalizeType(
	modules map[ast.QualifiedIdentifier]*parsed.Module, module *parsed.Module, t parsed.Type,
) normalized.Type {
	if t == nil {
		return nil //TODO: find places where it can happen and check there
	}
	normalize := func(x parsed.Type) normalized.Type {
		return normalizeType(modules, module, x)
	}
	switch t.(type) {
	case parsed.TFunc:
		{
			e := t.(parsed.TFunc)
			return normalized.TFunc{
				Location: e.Location,
				Params:   common.Map(normalize, e.Params),
				Return:   normalize(e.Return),
			}
		}
	case parsed.TRecord:
		{
			e := t.(parsed.TRecord)
			fields := map[ast.Identifier]normalized.Type{}
			for n, v := range e.Fields {
				fields[n] = normalize(v)
			}
			return normalized.TRecord{
				Location: e.Location,
				Fields:   fields,
			}
		}
	case parsed.TTuple:
		{
			e := t.(parsed.TTuple)
			return normalized.TTuple{
				Location: e.Location,
				Items:    common.Map(normalize, e.Items),
			}
		}
	case parsed.TUnit:
		{
			e := t.(parsed.TUnit)
			return normalized.TUnit{
				Location: e.Location,
			}
		}
	case parsed.TData:
		{
			e := t.(parsed.TData)
			return normalized.TData{
				Location: e.Location,
				Name:     e.Name,
				Args:     common.Map(normalize, e.Args),
			}
		}
	case parsed.TExternal:
		{
			e := t.(parsed.TExternal)
			return normalized.TExternal{
				Location: e.Location,
				Name:     e.Name,
				Args:     common.Map(normalize, e.Args),
			}
		}
	case parsed.TTypeParameter:
		{
			e := t.(parsed.TTypeParameter)
			return normalized.TTypeParameter{
				Location: e.Location,
				Name:     e.Name,
			}
		}
	case parsed.TNamed:
		{
			e := t.(parsed.TNamed)
			x, ok := findParsedType(modules, module, e.Name, e.Args)
			if !ok {
				panic(common.Error{Location: e.Location, Message: fmt.Sprintf("type `%s` not found", e.Name)})
			}
			return normalizeType(modules, module, x)
		}
	}
	panic(common.SystemError{Message: "impossible case"})
}

func findParsedDefinition(
	modules map[ast.QualifiedIdentifier]*parsed.Module, module *parsed.Module, name ast.QualifiedIdentifier,
) (*parsed.Module, parsed.Definition, bool) {
	var defNameEq = func(x parsed.Definition) bool {
		return ast.QualifiedIdentifier(x.Name) == name
	}

	if def, ok := common.Find(defNameEq, module.Definitions); ok {
		return module, def, true
	}

	ids := strings.Split(string(name), ".")
	defName := ast.QualifiedIdentifier(ids[len(ids)-1])

	for _, imp := range module.Imports {
		if slices.Contains(imp.Exposing, string(name)) {
			return findParsedDefinition(nil, modules[imp.ModuleIdentifier], defName)
		}
	}

	return nil, parsed.Definition{}, false
}

func findInfixFn(
	modules map[ast.QualifiedIdentifier]*parsed.Module, module *parsed.Module, name ast.InfixIdentifier,
) (*parsed.Module, parsed.Infix, bool) {
	var infNameEq = func(x parsed.Infix) bool { return x.Name == name }
	if inf, ok := common.Find(infNameEq, module.InfixFns); ok {
		return module, inf, true
	}

	for _, imp := range module.Imports {
		if slices.Contains(imp.Exposing, string(name)) {
			return findInfixFn(nil, modules[imp.ModuleIdentifier], name)
		}
	}
	return nil, parsed.Infix{}, false
}

func findParsedType(
	modules map[ast.QualifiedIdentifier]*parsed.Module,
	module *parsed.Module,
	name ast.QualifiedIdentifier,
	args []parsed.Type,
) (parsed.Type, bool) {
	var aliasNameEq = func(x parsed.Alias) bool {
		return ast.QualifiedIdentifier(x.Name) == name
	}

	if alias, ok := common.Find(aliasNameEq, module.Aliases); ok {
		if alias.Type == nil {
			return parsed.TExternal{
				Location: alias.Location,
				Name:     common.MakeExternalIdentifier(module.Name, alias.Name),
				Args:     args,
			}, true
		}
		return applyTypeArgs(alias.Type, args)
	}

	ids := strings.Split(string(name), ".")
	typeName := ast.QualifiedIdentifier(ids[len(ids)-1])

	for _, imp := range module.Imports {
		if slices.Contains(imp.Exposing, string(name)) {
			return findParsedType(nil, modules[imp.ModuleIdentifier], typeName, args)
		}
	}

	return nil, false
}

func applyTypeArgs(t parsed.Type, args []parsed.Type) (parsed.Type, bool) {
	switch t.(type) {
	case parsed.TFunc:
		return t, true
	case parsed.TRecord:
		return t, true
	case parsed.TTuple:
		return t, true
	case parsed.TUnit:
		return t, true
	case parsed.TData:
		{
			e := t.(parsed.TData)
			if len(e.Args) != len(args) {
				return nil, false
			}
			e.Args = args
			return e, true
		}
	case parsed.TNamed:
		{
			e := t.(parsed.TNamed)
			if len(e.Args) != len(args) {
				return nil, false
			}
			e.Args = args
			return e, true
		}
	case parsed.TExternal:
		{
			e := t.(parsed.TExternal)
			if len(e.Args) != len(args) {
				return nil, false
			}
			e.Args = args
			return e, true
		}
	case parsed.TTypeParameter:
		{
			return t, true
		}
	}
	panic(common.SystemError{Message: "impossible case"})
}

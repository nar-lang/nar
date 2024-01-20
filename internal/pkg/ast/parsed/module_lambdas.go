package parsed

import (
	"fmt"
	"maps"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
)

//TODO: organize this file

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

				let := normalized.LetMatch{
					ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
					Pattern: &normalized.PNamed{
						PatternBase: &normalized.PatternBase{Location: e.Location},
						Name:        replName,
					},
					Value:  replacement,
					Nested: def.Expression,
				}
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

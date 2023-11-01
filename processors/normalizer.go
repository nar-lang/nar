package processors

import (
	"fmt"
	"oak-compiler/ast"
	"oak-compiler/ast/normalized"
	"oak-compiler/ast/parsed"
	"oak-compiler/common"
	"slices"
	"strings"
)

func Normalize(path string, modules map[string]parsed.Module, normalizedModules map[string]normalized.Module) {
	if _, ok := normalizedModules[path]; ok {
		return
	}

	m := modules[path]

	for _, imp := range m.Imports {
		Normalize(imp.Path, modules, normalizedModules)
	}

	flattenDataTypes(&m)
	unwrapImports(&m, modules)
	modules[path] = m

	o := normalized.Module{
		Path:        m.Path,
		Definitions: map[ast.Identifier]normalized.Definition{},
	}

	for _, def := range m.Definitions {
		nDef := normalizeDefinition(modules, m, def)
		named, isNamed := def.Pattern.(parsed.PNamed)
		if !isNamed {
			panic(common.Error{
				Location: def.Pattern.GetLocation(),
				Message:  "only named pattern available at top level",
			})
		}
		o.Definitions[named.Name] = nDef
	}

	for _, imp := range m.Imports {
		o.DepPaths = append(o.DepPaths, imp.Path)
	}

	normalizedModules[m.Path] = o
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
				Values:   common.Map(func(x parsed.DataTypeValue) ast.Identifier { return x.Name }, it.Values),
			},
		})
		for _, value := range it.Values {
			var type_ parsed.Type = parsed.TExternal{
				Name: common.MakeExternalIdentifier(m.Name, it.Name),
				Args: typeArgs,
			}
			var body parsed.Expression = parsed.Constructor{
				Location:  value.Location,
				DataName:  common.MakeExternalIdentifier(m.Name, it.Name),
				ValueName: value.Name,
				Args: common.Map(
					func(i int) parsed.Expression {
						return parsed.Var{
							Location: value.Location,
							Name:     ast.QualifiedIdentifier(fmt.Sprintf("p%d", i)),
						}
					},
					common.Range(0, len(value.Params)),
				),
			}

			if len(value.Params) > 0 {
				body = parsed.Lambda{
					Params: common.Map(
						func(i int) parsed.Pattern {
							return parsed.PNamed{Location: value.Location, Name: ast.Identifier(fmt.Sprintf("p%d", i))}
						},
						common.Range(0, len(value.Params)),
					),
					Body: body,
				}
				type_ = parsed.TFunc{
					Params: value.Params,
					Return: type_,
				}
			}

			m.Definitions = append(m.Definitions, parsed.Definition{
				Pattern:    parsed.PNamed{Location: value.Location, Name: value.Name},
				Expression: body,
				Type:       type_,
			})
		}
	}
}

func unwrapImports(module *parsed.Module, modules map[string]parsed.Module) {
	for i, imp := range module.Imports {
		m := modules[imp.Path]
		modName := m.Name
		if imp.Alias != nil {
			modName = ast.QualifiedIdentifier(*imp.Alias)
		}

		var exp []string

		for _, d := range m.Definitions {
			n := string(d.Pattern.(parsed.PNamed).Name)
			if imp.ExposingAll || slices.Contains(imp.Exposing, n) {
				exp = append(exp, n)
			}
			exp = append(exp, fmt.Sprintf("%s.%s", modName, n))
		}

		for _, a := range m.Aliases {
			n := string(a.Name)
			if imp.ExposingAll || slices.Contains(imp.Exposing, n) {
				exp = append(exp, n)
				if dt, ok := a.Type.(parsed.TData); ok {
					for _, v := range dt.Values {
						exp = append(exp, string(v))
					}
				}
			}
			exp = append(exp, fmt.Sprintf("%s.%s", modName, n))
			if dt, ok := a.Type.(parsed.TData); ok {
				for _, v := range dt.Values {
					exp = append(exp, fmt.Sprintf("%s.%s", modName, v))
				}
			}
		}

		for _, a := range m.InfixFns {
			n := string(a.Name)
			if imp.ExposingAll || slices.Contains(imp.Exposing, n) {
				exp = append(exp, n)
			}
			exp = append(exp, fmt.Sprintf("%s.%s", modName, n))
		}
		imp.Exposing = exp
		module.Imports[i] = imp
	}
}

var nextDefinitionId = uint64(0)

func normalizeDefinition(modules map[string]parsed.Module, module parsed.Module, def parsed.Definition) normalized.Definition {
	nextDefinitionId++
	o := normalized.Definition{
		Id: nextDefinitionId,
	}
	o.Pattern = normalizePattern(modules, module, def.Pattern)
	o.Expression = normalizeExpression(modules, module, def.Expression)
	o.Type = normalizeType(modules, module, def.Type)
	return o
}

func normalizePattern(modules map[string]parsed.Module, module parsed.Module, pattern parsed.Pattern) normalized.Pattern {
	normalize := func(p parsed.Pattern) normalized.Pattern { return normalizePattern(modules, module, p) }

	switch pattern.(type) {
	case parsed.PAlias:
		{
			e := pattern.(parsed.PAlias)
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
	case parsed.PDataValue:
		{
			e := pattern.(parsed.PDataValue)
			mod, def, ok := findParsedDefinition(modules, module, e.Name)
			if !ok {
				findParsedDefinition(modules, module, e.Name)
				panic(common.Error{Location: e.Location, Message: "data constructor not found"})
			}
			return normalized.PDataValue{
				Location:       e.Location,
				Type:           normalizeType(modules, module, e.Type),
				ModulePath:     mod.Path,
				DefinitionName: def.Pattern.(parsed.PNamed).Name,
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

func normalizeExpression(modules map[string]parsed.Module, module parsed.Module, expr parsed.Expression) normalized.Expression {
	if expr == nil {
		return nil
	}

	normalize := func(e parsed.Expression) normalized.Expression {
		return normalizeExpression(modules, module, e)
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
	case parsed.Call:
		{
			e := expr.(parsed.Call)
			return normalized.Call{
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
				Location:  e.Location,
				DataName:  e.DataName,
				ValueName: e.ValueName,
				Args:      common.Map(normalize, e.Args),
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
	case parsed.Let:
		{
			e := expr.(parsed.Let)
			return normalized.Let{
				Location:   e.Location,
				Definition: normalizeDefinition(modules, module, e.Definition),
				Body:       normalize(e.Body),
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
					return normalized.SelectCase{
						Location:   e.Location,
						Pattern:    normalizePattern(modules, module, i.Pattern),
						Expression: normalize(i.Expression),
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
			if m, d, ok := findParsedDefinition(modules, module, e.RecordName); !ok {
				return normalized.UpdateGlobal{
					Location:       e.Location,
					ModulePath:     m.Path,
					DefinitionName: d.Pattern.(parsed.PNamed).Name,
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
				Params: common.Map(func(p parsed.Pattern) normalized.Pattern {
					return normalizePattern(modules, module, p)
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
						if o2.Fn.Precedence > o1.Fn.Precedence || (o2.Fn.Precedence == o1.Fn.Precedence && o1.Fn.Associativity == parsed.Left) {
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

					return normalized.Call{
						Location: e.Location,
						Func: normalized.Global{
							Location:       e.Location,
							ModulePath:     m.Path,
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
			if m, d, ok := findParsedDefinition(modules, module, e.Name); ok {
				return normalized.Global{
					Location:       e.Location,
					ModulePath:     m.Path,
					DefinitionName: d.Pattern.(parsed.PNamed).Name,
				}
			}
			return normalized.Local{
				Location: e.Location,
				Name:     ast.Identifier(e.Name),
			}
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
					ModulePath:     m.Path,
					DefinitionName: d.Pattern.(parsed.PNamed).Name,
				}
			}
		}
	}
	panic(common.SystemError{Message: "impossible case"})
}

func normalizeType(modules map[string]parsed.Module, module parsed.Module, t parsed.Type) normalized.Type {
	if t == nil {
		return nil
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
				panic(common.Error{Location: e.Location, Message: "type not found"})
			}
			return normalizeType(modules, module, x)
		}
	}
	panic(common.SystemError{Message: "impossible case"})
}

func findParsedDefinition(
	modules map[string]parsed.Module, module parsed.Module, name ast.QualifiedIdentifier,
) (parsed.Module, parsed.Definition, bool) {
	var defNameEq = func(x parsed.Definition) bool {
		n, ok := x.Pattern.(parsed.PNamed)
		return ok && ast.QualifiedIdentifier(n.Name) == name
	}

	if def, ok := common.Find(defNameEq, module.Definitions); ok {
		return module, def, true
	}

	ids := strings.Split(string(name), ".")
	defName := ast.QualifiedIdentifier(ids[len(ids)-1])

	for _, imp := range module.Imports {
		if slices.Contains(imp.Exposing, string(name)) {
			return findParsedDefinition(nil, modules[imp.Path], defName)
		}
	}

	return parsed.Module{}, parsed.Definition{}, false
}

func findInfixFn(modules map[string]parsed.Module, module parsed.Module, name ast.InfixIdentifier) (parsed.Module, parsed.Infix, bool) {
	var infNameEq = func(x parsed.Infix) bool { return x.Name == name }
	if inf, ok := common.Find(infNameEq, module.InfixFns); ok {
		return module, inf, true
	}

	for _, imp := range module.Imports {
		if slices.Contains(imp.Exposing, string(name)) {
			return findInfixFn(nil, modules[imp.Path], name)
		}
	}
	return parsed.Module{}, parsed.Infix{}, false
}

func findParsedType(
	modules map[string]parsed.Module, module parsed.Module, name ast.QualifiedIdentifier, args []parsed.Type,
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
			return findParsedType(nil, modules[imp.Path], typeName, args)
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

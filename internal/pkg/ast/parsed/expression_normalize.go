package parsed

import (
	"fmt"
	"maps"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
	"strings"
)

func normalizeExpression(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) func(expr Expression) (normalized.Expression, error) {
	return func(expr Expression) (normalized.Expression, error) {
		if expr == nil {
			return nil, nil
		}
		normalizedExpression, err := expr.normalize(locals, modules, module, normalizedModule)
		if err != nil {
			return nil, err
		}
		expr.setSuccessor(normalizedExpression)
		return normalizedExpression, nil
	}
}

func newAmbiguousInfixError(ids []ast.FullIdentifier, name ast.InfixIdentifier, loc ast.Location) error {
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

func newAmbiguousDefinitionError(ids []ast.FullIdentifier, name ast.QualifiedIdentifier, loc ast.Location) error {
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

func (e *Access) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	record, err := normalize(e.record)
	if err != nil {
		return nil, err
	}
	return normalized.NewAccess(e.location, record, e.fieldName), nil
}

func (e *Apply) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	fn, err := normalize(e.func_)
	if err != nil {
		return nil, err
	}
	args, err := common.MapError(normalize, e.args)
	if err != nil {
		return nil, err
	}
	return normalized.NewApply(e.location, fn, args), nil
}

func (e *Const) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	return normalized.NewConst(e.location, e.value), nil
}

func (e *Constructor) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	args, err := common.MapError(normalize, e.args)
	if err != nil {
		return nil, err
	}
	return normalized.NewConstructor(e.location, e.moduleName, e.dataName, e.optionName, args), nil
}

func (e *If) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	boolType := normalized.NewTData(
		e.condition.GetLocation(),
		common.NarBaseBasicsBool,
		nil,
		[]*normalized.DataOption{
			normalized.NewDataOption(common.NarBaseBasicsTrueName, false, nil),
			normalized.NewDataOption(common.NarBaseBasicsFalseName, false, nil),
		},
	)
	condition, err := normalize(e.condition)
	if err != nil {
		return nil, err
	}
	positive, err := normalize(e.positive)
	if err != nil {
		return nil, err
	}
	negative, err := normalize(e.negative)
	if err != nil {
		return nil, err
	}
	return normalized.NewSelect(
		e.location,
		condition,
		[]*normalized.SelectCase{
			normalized.NewSelectCase(
				e.positive.GetLocation(),
				normalized.NewPOption(
					e.positive.GetLocation(), boolType, common.NarBaseBasicsName, common.NarBaseBasicsTrueName, nil),
				positive),
			normalized.NewSelectCase(
				e.negative.GetLocation(),
				normalized.NewPOption(
					e.negative.GetLocation(), boolType, common.NarBaseBasicsName, common.NarBaseBasicsFalseName, nil),
				negative),
		}), nil
}

func (e *LetMatch) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	innerLocals := maps.Clone(locals)
	pattern, err := normalizePattern(innerLocals, modules, module, normalizedModule)(e.pattern)
	if err != nil {
		return nil, err
	}
	value, err := normalizeExpression(innerLocals, modules, module, normalizedModule)(e.value)
	if err != nil {
		return nil, err
	}
	nested, err := normalizeExpression(innerLocals, modules, module, normalizedModule)(e.nested)
	if err != nil {
		return nil, err
	}
	return normalized.NewLet(e.location, pattern, value, nested), nil
}

func (e *LetDef) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	innerLocals := maps.Clone(locals)
	innerLocals[e.name] = normalized.NewPNamed(e.nameLocation, nil, e.name)
	params, err := common.MapError(normalizePattern(innerLocals, modules, module, normalizedModule), e.params)
	if err != nil {
		return nil, err
	}
	body, err := normalizeExpression(innerLocals, modules, module, normalizedModule)(e.body)
	if err != nil {
		return nil, err
	}
	nested, err := normalizeExpression(innerLocals, modules, module, normalizedModule)(e.nested)
	if err != nil {
		return nil, err
	}
	type_, err := normalizeType(modules, module, nil, nil)(e.fnType)
	if err != nil {
		return nil, err
	}
	return normalized.NewFunction(e.location, e.name, params, body, type_, nested), nil
}

func (e *List) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	items, err := common.MapError(normalize, e.items)
	if err != nil {
		return nil, err
	}
	return normalized.NewList(e.location, items), nil
}

func (e *NativeCall) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	args, err := common.MapError(normalize, e.args)
	if err != nil {
		return nil, err
	}
	return normalized.NewNativeCall(e.location, e.name, args), nil
}

func (e *Record) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	fields, err := common.MapError(func(i RecordField) (*normalized.RecordField, error) {
		value, err := normalize(i.Value)
		if err != nil {
			return nil, err
		}
		return normalized.NewRecordField(i.Location, i.Name, value), nil
	}, e.fields)
	if err != nil {
		return nil, err
	}
	return normalized.NewRecord(e.location, fields), nil
}

func (e *Select) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	condition, err := normalize(e.condition)
	if err != nil {
		return nil, err
	}
	cases, err := common.MapError(func(i SelectCase) (*normalized.SelectCase, error) {
		innerLocals := maps.Clone(locals)
		pattern, err := normalizePattern(innerLocals, modules, module, normalizedModule)(i.Pattern)
		if err != nil {
			return nil, err
		}
		expression, err := normalizeExpression(innerLocals, modules, module, normalizedModule)(i.Expression)
		if err != nil {
			return nil, err
		}
		return normalized.NewSelectCase(i.Location, pattern, expression), nil
	}, e.cases)
	if err != nil {
		return nil, err
	}
	return normalized.NewSelect(e.location, condition, cases), nil
}

func (e *Tuple) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	items, err := common.MapError(normalize, e.items)
	if err != nil {
		return nil, err
	}
	return normalized.NewTuple(e.location, items), nil
}

func (e *Update) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	d, m, ids := findParsedDefinition(modules, module, e.recordName, normalizedModule)
	fields, err := common.MapError(func(i RecordField) (*normalized.RecordField, error) {
		value, err := normalize(i.Value)
		if err != nil {
			return nil, err
		}
		return normalized.NewRecordField(i.Location, i.Name, value), nil
	}, e.fields)
	if err != nil {
		return nil, err
	}

	if len(ids) == 1 {
		return normalized.NewUpdateGlobal(e.location, m.name, d.name, fields), nil
	} else if len(ids) > 1 {
		return nil, newAmbiguousDefinitionError(ids, e.recordName, e.location)
	}

	if lc, ok := locals[ast.Identifier(e.recordName)]; ok {
		return normalized.NewUpdateLocal(e.location, ast.Identifier(e.recordName), lc, fields), nil
	} else {
		return nil, common.Error{
			Location: e.location,
			Message:  fmt.Sprintf("identifier `%s` not found", e.location.Text()),
		}
	}
}

func (e *Lambda) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	params, err := common.MapError(normalizePattern(locals, modules, module, normalizedModule), e.params)
	if err != nil {
		return nil, err
	}
	body, err := normalize(e.body)
	if err != nil {
		return nil, err
	}
	return normalized.NewLambda(e.location, params, body), nil
}

func (e *Accessor) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	return normalize(NewLambda(e.location,
		[]Pattern{NewPNamed(e.location, "x")},
		nil,
		NewAccess(e.location, NewVar(e.location, "x"), e.fieldName)))
}

func (e *BinOp) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	var output []BinOpItem
	var operators []BinOpItem
	for _, o1 := range e.items {
		if o1.Expression != nil {
			output = append(output, o1)
		} else {
			if infixFn, _, ids := findParsedInfixFn(modules, module, o1.Infix); len(ids) != 1 {
				return nil, newAmbiguousInfixError(ids, o1.Infix, e.location)
			} else {
				o1.Fn = infixFn
			}

			for i := len(operators) - 1; i >= 0; i-- {
				o2 := operators[i]
				if o2.Fn.precedence > o1.Fn.precedence ||
					(o2.Fn.precedence == o1.Fn.precedence && o1.Fn.associativity == Left) {
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
			return nil, newAmbiguousInfixError(ids, op, e.location)
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

			return normalized.NewApply(
				e.location,
				normalized.NewGlobal(e.location, m.name, infixA.alias),
				[]normalized.Expression{left, right},
			), nil
		}
	}

	return buildTree()
}

func (e *Negate) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	nested, err := normalize(e.nested)
	if err != nil {
		return nil, err
	}
	return normalized.NewApply(
		e.location,
		normalized.NewGlobal(e.location, common.NarBaseMathName, common.NarBaseMathNegName),
		[]normalized.Expression{nested},
	), nil
}

func (e *Var) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	if lc, ok := locals[ast.Identifier(e.name)]; ok {
		return normalized.NewLocal(e.location, ast.Identifier(e.name), lc), nil
	}

	d, m, ids := findParsedDefinition(modules, module, e.name, normalizedModule)
	if len(ids) == 1 {
		return normalized.NewGlobal(e.location, m.name, d.name), nil
	} else if len(ids) > 1 {
		return nil, newAmbiguousDefinitionError(ids, e.name, e.location)
	}

	parts := strings.Split(string(e.name), ".")
	if len(parts) > 1 {
		varAccess := NewVar(e.location, ast.QualifiedIdentifier(parts[0]))
		for i := 1; i < len(parts); i++ {
			varAccess = NewAccess(e.location, varAccess, ast.Identifier(parts[i]))
		}
		return normalize(varAccess)
	}

	return nil, common.Error{
		Location: e.location,
		Message:  fmt.Sprintf("identifier `%s` not found", e.location.Text()),
	}
}

func (e *InfixVar) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	if i, m, ids := findParsedInfixFn(modules, module, e.infix); len(ids) != 1 {
		return nil, newAmbiguousInfixError(ids, e.infix, e.location)
	} else if d, _, ids := findParsedDefinition(nil, m, ast.QualifiedIdentifier(i.alias), normalizedModule); len(ids) != 1 {
		return nil, newAmbiguousDefinitionError(ids, ast.QualifiedIdentifier(i.alias), e.location)
	} else {
		return normalized.NewGlobal(e.location, m.name, d.name), nil
	}
}

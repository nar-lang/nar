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
	record, err := normalize(e.Record)
	if err != nil {
		return nil, err
	}
	return normalized.Access{
		ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
		Record:         record,
		FieldName:      e.FieldName,
	}, nil
}

func (e *Apply) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	fn, err := normalize(e.Func)
	if err != nil {
		return nil, err
	}
	args, err := common.MapError(normalize, e.Args)
	if err != nil {
		return nil, err
	}
	return normalized.Apply{
		ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
		Func:           fn,
		Args:           args,
	}, nil
}

func (e *Const) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	return normalized.Const{
		ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
		Value:          e.Value,
	}, nil
}

func (e *Constructor) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	args, err := common.MapError(normalize, e.Args)
	if err != nil {
		return nil, err
	}
	return normalized.Constructor{
		ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
		ModuleName:     e.ModuleName,
		DataName:       e.DataName,
		OptionName:     e.OptionName,
		Args:           args,
	}, nil
}

func (e *If) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	boolType := &normalized.TData{
		Location: e.Condition.GetLocation(),
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
	return normalized.Select{
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
	}, nil
}

func (e *LetMatch) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	innerLocals := maps.Clone(locals)
	pattern, err := normalizePattern(innerLocals, modules, module, normalizedModule)(e.Pattern)
	if err != nil {
		return nil, err
	}
	value, err := normalizeExpression(innerLocals, modules, module, normalizedModule)(e.Value)
	if err != nil {
		return nil, err
	}
	nested, err := normalizeExpression(innerLocals, modules, module, normalizedModule)(e.Nested)
	if err != nil {
		return nil, err
	}
	return normalized.LetMatch{
		ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
		Pattern:        pattern,
		Value:          value,
		Nested:         nested,
	}, nil
}

func (e *LetDef) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	innerLocals := maps.Clone(locals)
	innerLocals[e.Name] = &normalized.PNamed{
		PatternBase: &normalized.PatternBase{Location: e.NameLocation},
		Name:        e.Name,
	}
	params, err := common.MapError(normalizePattern(innerLocals, modules, module, normalizedModule), e.Params)
	if err != nil {
		return nil, err
	}
	body, err := normalizeExpression(innerLocals, modules, module, normalizedModule)(e.Body)
	if err != nil {
		return nil, err
	}
	nested, err := normalizeExpression(innerLocals, modules, module, normalizedModule)(e.Nested)
	if err != nil {
		return nil, err
	}
	type_, err := normalizeType(modules, module, nil, nil)(e.FnType)
	if err != nil {
		return nil, err
	}
	return normalized.LetDef{
		ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
		Name:           e.Name,
		Params:         params,
		FnType:         type_,
		Body:           body,
		Nested:         nested,
	}, nil
}

func (e *List) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	items, err := common.MapError(normalize, e.Items)
	if err != nil {
		return nil, err
	}
	return normalized.List{
		ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
		Items:          items,
	}, nil
}

func (e *NativeCall) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	args, err := common.MapError(normalize, e.Args)
	if err != nil {
		return nil, err
	}
	return normalized.NativeCall{
		ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
		Name:           e.Name,
		Args:           args,
	}, nil
}

func (e *Record) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	fields, err := common.MapError(func(i RecordField) (normalized.RecordField, error) {
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
	return normalized.Record{
		ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
		Fields:         fields,
	}, nil
}

func (e *Select) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	condition, err := normalize(e.Condition)
	if err != nil {
		return nil, err
	}
	cases, err := common.MapError(func(i SelectCase) (normalized.SelectCase, error) {
		innerLocals := maps.Clone(locals)
		pattern, err := normalizePattern(innerLocals, modules, module, normalizedModule)(i.Pattern)
		if err != nil {
			return normalized.SelectCase{}, err
		}
		expression, err := normalizeExpression(innerLocals, modules, module, normalizedModule)(i.Expression)
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
	return normalized.Select{
		ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
		Condition:      condition,
		Cases:          cases,
	}, nil
}

func (e *Tuple) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	items, err := common.MapError(normalize, e.Items)
	if err != nil {
		return nil, err
	}
	return normalized.Tuple{
		ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
		Items:          items,
	}, nil
}

func (e *Update) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	d, m, ids := findParsedDefinition(modules, module, e.RecordName, normalizedModule)
	fields, err := common.MapError(func(i RecordField) (normalized.RecordField, error) {
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

		return normalized.UpdateGlobal{
			ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
			ModuleName:     m.name,
			DefinitionName: d.Name,
			Fields:         fields,
		}, nil
	} else if len(ids) > 1 {
		return nil, newAmbiguousDefinitionError(ids, e.RecordName, e.Location)
	}

	return normalized.UpdateLocal{
		ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
		RecordName:     ast.Identifier(e.RecordName),
		Fields:         fields,
	}, nil
}

func (e *Lambda) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	params, err := common.MapError(normalizePattern(locals, modules, module, normalizedModule), e.Params)
	if err != nil {
		return nil, err
	}
	body, err := normalize(e.Body)
	if err != nil {
		return nil, err
	}
	return normalized.Lambda{
		ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
		Params:         params,
		Body:           body,
	}, nil
}

func (e *Accessor) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	return normalize(&Lambda{
		Params: []Pattern{NewPNamed(e.Location, "x")},
		Body: &Access{
			ExpressionBase: &ExpressionBase{
				Location: e.Location,
			},
			Record: &Var{
				ExpressionBase: &ExpressionBase{
					Location: e.Location,
				},
				Name: "x",
			},
			FieldName: e.FieldName,
		},
	})
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

			return normalized.Apply{
				ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
				Func: normalized.Global{
					ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
					ModuleName:     m.name,
					DefinitionName: infixA.alias,
				},
				Args: []normalized.Expression{left, right},
			}, nil
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
	nested, err := normalize(e.Nested)
	if err != nil {
		return nil, err
	}
	return normalized.Apply{
		ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
		Func: normalized.Global{
			ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
			ModuleName:     common.NarCoreMath,
			DefinitionName: common.NarCoreMathNeg,
		},
		Args: []normalized.Expression{nested},
	}, nil
}

func (e *Var) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	normalize := normalizeExpression(locals, modules, module, normalizedModule)
	if lc, ok := locals[ast.Identifier(e.Name)]; ok {
		return normalized.Local{
			ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
			Name:           ast.Identifier(e.Name),
			Target:         lc,
		}, nil
	}

	d, m, ids := findParsedDefinition(modules, module, e.Name, normalizedModule)
	if len(ids) == 1 {
		return normalized.Global{
			ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
			ModuleName:     m.name,
			DefinitionName: d.Name,
		}, nil
	} else if len(ids) > 1 {
		return nil, newAmbiguousDefinitionError(ids, e.Name, e.Location)
	}

	parts := strings.Split(string(e.Name), ".")
	if len(parts) > 1 {
		varAccess := Expression(&Var{
			ExpressionBase: &ExpressionBase{
				Location: e.Location,
			},
			Name: ast.QualifiedIdentifier(parts[0]),
		})
		for i := 1; i < len(parts); i++ {
			varAccess = &Access{
				ExpressionBase: &ExpressionBase{
					Location: e.Location,
				},
				Record:    varAccess,
				FieldName: ast.Identifier(parts[i]),
			}
		}
		return normalize(varAccess)
	}

	return nil, common.Error{
		Location: e.Location,
		Message:  fmt.Sprintf("identifier `%s` not found", e.Location.Text()),
	}
}

func (e *InfixVar) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	if i, m, ids := findParsedInfixFn(modules, module, e.Infix); len(ids) != 1 {
		return nil, newAmbiguousInfixError(ids, e.Infix, e.Location)
	} else if d, _, ids := findParsedDefinition(nil, m, ast.QualifiedIdentifier(i.alias), normalizedModule); len(ids) != 1 {
		return nil, newAmbiguousDefinitionError(ids, ast.QualifiedIdentifier(i.alias), e.Location)
	} else {
		return normalized.Global{
			ExpressionBase: &normalized.ExpressionBase{Location: e.Location},
			ModuleName:     m.name,
			DefinitionName: d.Name,
		}, nil
	}
}

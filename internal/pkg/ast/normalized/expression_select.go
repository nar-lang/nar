package normalized

import (
	"maps"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
)

type Select struct {
	*expressionBase
	condition Expression
	cases     []*SelectCase
}

func NewSelect(loc ast.Location, condition Expression, cases []*SelectCase) Expression {
	return &Select{
		expressionBase: newExpressionBase(loc),
		condition:      condition,
		cases:          cases,
	}
}

func (e *Select) flattenLambdas(parentName ast.Identifier, m *Module, locals map[ast.Identifier]Pattern) Expression {
	e.condition = e.condition.flattenLambdas(parentName, m, locals)
	for i, a := range e.cases {
		innerLocals := maps.Clone(locals)
		a.pattern.extractLocals(innerLocals)
		e.cases[i].expression = a.expression.flattenLambdas(parentName, m, innerLocals)
	}
	return e
}

func (e *Select) replaceLocals(replace map[ast.Identifier]Expression) Expression {
	e.condition = e.condition.replaceLocals(replace)
	for i, a := range e.cases {
		e.cases[i].expression = a.expression.replaceLocals(replace)
	}
	return e
}

func (e *Select) extractUsedLocalsSet(definedLocals map[ast.Identifier]Pattern, usedLocals map[ast.Identifier]struct{}) {
	e.condition.extractUsedLocalsSet(definedLocals, usedLocals)
	for _, c := range e.cases {
		c.expression.extractUsedLocalsSet(definedLocals, usedLocals)
	}
}

func (e *Select) annotate(ctx *typed.SolvingContext, typeParams typeParamsMap, modules map[ast.QualifiedIdentifier]*Module, typedModules map[ast.QualifiedIdentifier]*typed.Module, moduleName ast.QualifiedIdentifier, stack []*typed.Definition) (typed.Expression, error) {
	condition, err := e.condition.annotate(ctx, typeParams, modules, typedModules, moduleName, stack)
	if err != nil {
		return nil, err
	}
	cases, err := common.MapError(func(c *SelectCase) (*typed.SelectCase, error) {
		localTypeParams := maps.Clone(typeParams)
		pattern, err := c.pattern.annotate(ctx, localTypeParams, modules, typedModules, moduleName, false, stack)
		if err != nil {
			return nil, err
		}
		expr, err := c.expression.annotate(ctx, localTypeParams, modules, typedModules, moduleName, stack)
		if err != nil {
			return nil, err
		}
		return typed.NewSelectCase(c.location, pattern, expr), nil
	}, e.cases)
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(typed.NewSelect(ctx, e.location, condition, cases))
}

func (e *Select) Cases() []*SelectCase {
	return e.cases
}

type SelectCase struct {
	location   ast.Location
	pattern    Pattern
	expression Expression
}

func (c SelectCase) Location() ast.Location {
	return c.location
}

func (c SelectCase) Pattern() Pattern {
	return c.pattern
}

func NewSelectCase(loc ast.Location, pattern Pattern, expression Expression) *SelectCase {
	return &SelectCase{
		location:   loc,
		pattern:    pattern,
		expression: expression,
	}
}

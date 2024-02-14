package normalized

import (
	"maps"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
)

type Let struct {
	*expressionBase
	pattern Pattern
	value   Expression
	nested  Expression
}

func NewLet(loc ast.Location, pattern Pattern, value Expression, nested Expression) Expression {
	return &Let{
		expressionBase: newExpressionBase(loc),
		pattern:        pattern,
		value:          value,
		nested:         nested,
	}
}

func (e *Let) flattenLambdas(parentName ast.Identifier, m *Module, locals map[ast.Identifier]Pattern) Expression {
	innerLocals := maps.Clone(locals)
	e.pattern.extractLocals(innerLocals)
	e.value = e.value.flattenLambdas(parentName, m, innerLocals)
	e.nested = e.nested.flattenLambdas(parentName, m, innerLocals)
	return e
}

func (e *Let) replaceLocals(replace map[ast.Identifier]Expression) Expression {
	e.value = e.value.replaceLocals(replace)
	e.nested = e.nested.replaceLocals(replace)
	return e
}

func (e *Let) extractUsedLocalsSet(definedLocals map[ast.Identifier]Pattern, usedLocals map[ast.Identifier]struct{}) {
	e.value.extractUsedLocalsSet(definedLocals, usedLocals)
	e.nested.extractUsedLocalsSet(definedLocals, usedLocals)
}

func (e *Let) annotate(ctx *typed.SolvingContext, typeParams typeParamsMap, modules map[ast.QualifiedIdentifier]*Module, typedModules map[ast.QualifiedIdentifier]*typed.Module, moduleName ast.QualifiedIdentifier, stack []*typed.Definition) (typed.Expression, error) {
	localTypeParams := maps.Clone(typeParams)

	pattern, err := e.pattern.annotate(ctx, localTypeParams, modules, typedModules, moduleName, true, stack)
	if err != nil {
		return nil, err
	}
	value, err := e.value.annotate(ctx, localTypeParams, modules, typedModules, moduleName, stack)
	if err != nil {
		return nil, err
	}
	body, err := e.nested.annotate(ctx, localTypeParams, modules, typedModules, moduleName, stack)
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(typed.NewLet(ctx, e.location, pattern, value, body))
}

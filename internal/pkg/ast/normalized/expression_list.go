package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
)

type List struct {
	*expressionBase
	items []Expression
}

func NewList(loc ast.Location, items []Expression) Expression {
	return &List{
		expressionBase: newExpressionBase(loc),
		items:          items,
	}
}

func (e *List) flattenLambdas(parentName ast.Identifier, m *Module, locals map[ast.Identifier]Pattern) Expression {
	for i, a := range e.items {
		e.items[i] = a.flattenLambdas(parentName, m, locals)
	}
	return e
}

func (e *List) replaceLocals(replace map[ast.Identifier]Expression) Expression {
	for i, a := range e.items {
		e.items[i] = a.replaceLocals(replace)
	}
	return e
}

func (e *List) extractUsedLocalsSet(definedLocals map[ast.Identifier]Pattern, usedLocals map[ast.Identifier]struct{}) {
	for _, i := range e.items {
		i.extractUsedLocalsSet(definedLocals, usedLocals)
	}
}

func (e *List) annotate(ctx *typed.SolvingContext, typeParams typeParamsMap, modules map[ast.QualifiedIdentifier]*Module, typedModules map[ast.QualifiedIdentifier]*typed.Module, moduleName ast.QualifiedIdentifier, stack []*typed.Definition) (typed.Expression, error) {
	items, err := common.MapError(func(e Expression) (typed.Expression, error) {
		return e.annotate(ctx, typeParams, modules, typedModules, moduleName, stack)
	}, e.items)
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(typed.NewList(ctx, e.location, items))
}

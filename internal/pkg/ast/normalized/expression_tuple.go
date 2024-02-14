package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
)

type Tuple struct {
	*expressionBase
	items []Expression
}

func NewTuple(loc ast.Location, items []Expression) Expression {
	return &Tuple{
		expressionBase: newExpressionBase(loc),
		items:          items,
	}
}

func (e *Tuple) flattenLambdas(parentName ast.Identifier, m *Module, locals map[ast.Identifier]Pattern) Expression {
	for i, a := range e.items {
		e.items[i] = a.flattenLambdas(parentName, m, locals)
	}
	return e
}

func (e *Tuple) replaceLocals(replace map[ast.Identifier]Expression) Expression {
	for i, a := range e.items {
		e.items[i] = a.replaceLocals(replace)
	}
	return e
}

func (e *Tuple) extractUsedLocalsSet(definedLocals map[ast.Identifier]Pattern, usedLocals map[ast.Identifier]struct{}) {
	for _, i := range e.items {
		i.extractUsedLocalsSet(definedLocals, usedLocals)
	}
}

func (e *Tuple) annotate(ctx *typed.SolvingContext, typeParams typeParamsMap, modules map[ast.QualifiedIdentifier]*Module, typedModules map[ast.QualifiedIdentifier]*typed.Module, moduleName ast.QualifiedIdentifier, stack []*typed.Definition) (typed.Expression, error) {
	items, err := common.MapError(func(e Expression) (typed.Expression, error) {
		return e.annotate(ctx, typeParams, modules, typedModules, moduleName, stack)
	}, e.items)
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(typed.NewTuple(ctx, e.location, items))
}

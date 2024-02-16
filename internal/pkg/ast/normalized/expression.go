package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
)

type Expression interface {
	Statement
	_expression()
	flattenLambdas(parentName ast.Identifier, m *Module, locals map[ast.Identifier]Pattern) Expression
	replaceLocals(replace map[ast.Identifier]Expression) Expression
	extractUsedLocalsSet(definedLocals map[ast.Identifier]Pattern, usedLocals map[ast.Identifier]struct{})
	annotate(ctx *typed.SolvingContext, typeParams typeParamsMap, modules map[ast.QualifiedIdentifier]*Module, typedModules map[ast.QualifiedIdentifier]*typed.Module, moduleName ast.QualifiedIdentifier, stack []*typed.Definition) (typed.Expression, error)
}

type expressionBase struct {
	location  ast.Location
	successor typed.Expression
}

func newExpressionBase(loc ast.Location) *expressionBase {
	return &expressionBase{
		location: loc,
	}
}

func (e *expressionBase) _expression() {}

func (e *expressionBase) Location() ast.Location {
	return e.location
}

func (e *expressionBase) Successor() typed.Statement {
	return e.successor
}

func (e *expressionBase) setSuccessor(typedExpression typed.Expression) (typed.Expression, error) {
	e.successor = typedExpression
	return typedExpression, nil
}

package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

type Expression interface {
	Statement
	normalize(
		locals map[ast.Identifier]normalized.Pattern,
		modules map[ast.QualifiedIdentifier]*Module,
		module *Module,
		normalizedModule *normalized.Module,
	) (normalized.Expression, error)
	Successor() normalized.Expression
	setSuccessor(expr normalized.Expression) (normalized.Expression, error)
}

type expressionBase struct {
	location  ast.Location
	successor normalized.Expression
}

func (*expressionBase) _parsed() {}

func (e *expressionBase) GetLocation() ast.Location {
	return e.location
}

func (e *expressionBase) Successor() normalized.Expression {
	return e.successor
}

func (e *expressionBase) setSuccessor(expr normalized.Expression) (normalized.Expression, error) {
	e.successor = expr
	return expr, nil
}

func newExpressionBase(location ast.Location) *expressionBase {
	return &expressionBase{location: location}
}

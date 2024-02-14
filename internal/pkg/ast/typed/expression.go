package typed

import (
	"nar-compiler/internal/pkg/ast"
)

type Expression interface {
	Statement
	bytecoder
	_expression()
	Type() Type
	appendEquations(eqs Equations, loc *ast.Location, localDefs localTypesMap, ctx *SolvingContext, stack []*Definition) (Equations, error)
	mapTypes(subst map[uint64]Type) error
	checkPatterns() error
	setAnnotation(annotatedType *TUnbound)
}

type expressionBase struct {
	location ast.Location
	type_    Type
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

func (e *expressionBase) Type() Type {
	return e.type_
}

func (e *expressionBase) Children() []Statement {
	return []Statement{e.type_}
}

func (e *expressionBase) setAnnotation(type_ *TUnbound) {
	e.type_ = type_
}

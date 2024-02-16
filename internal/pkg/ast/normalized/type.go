package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
)

type Type interface {
	Statement
	_type()
	annotate(ctx *typed.SolvingContext, params typeParamsMap, source bool, placeholders placeholderMap) (typed.Type, error)
	SetSuccessor(typedType typed.Type) typed.Type
}

type typeBase struct {
	location  ast.Location
	successor typed.Type
}

func newTypeBase(loc ast.Location) *typeBase {
	return &typeBase{location: loc}
}

func (t *typeBase) _type() {}

func (t *typeBase) Location() ast.Location {
	return t.location
}

func (t *typeBase) Successor() typed.Statement {
	return t.successor
}

func (t *typeBase) setSuccessor(typedType typed.Type) (typed.Type, error) {
	t.successor = typedType
	return typedType, nil
}

func (t *typeBase) SetSuccessor(typedType typed.Type) typed.Type {
	_, _ = t.setSuccessor(typedType)
	return typedType
}

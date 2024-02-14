package typed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/bytecode"
)

type Type interface {
	Statement
	bytecoder
	_type()
	equalsTo(other Type, req map[ast.FullIdentifier]struct{}) bool
	merge(other Type, loc ast.Location) (Equations, error)
	mapTo(subst map[uint64]Type) (Type, error)
	makeUnique(ctx *SolvingContext, ubMap map[uint64]uint64) Type
}

type typeBase struct {
	location ast.Location
}

func newTypeBase(loc ast.Location) *typeBase {
	return &typeBase{
		location: loc,
	}
}

func (t *typeBase) _type() {}

func (t *typeBase) Location() ast.Location {
	return t.location
}

func (t *typeBase) appendBytecode(ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary) ([]bytecode.Op, []ast.Location) {
	return nil, nil
}

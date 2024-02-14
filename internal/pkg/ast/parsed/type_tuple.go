package parsed

import "nar-compiler/internal/pkg/ast"

type TTuple struct {
	*typeBase
	items []Type
}

func NewTTuple(loc ast.Location, items []Type) Type {
	return &TTuple{
		typeBase: newTypeBase(loc),
		items:    items,
	}
}

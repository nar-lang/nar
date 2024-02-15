package parsed

import (
	"nar-compiler/internal/pkg/ast"
)

func NewTTuple(loc ast.Location, items []Type) Type {
	return &TTuple{
		typeBase: newTypeBase(loc),
		items:    items,
	}
}

type TTuple struct {
	*typeBase
	items []Type
}

func (t *TTuple) Iterate(f func(statement Statement)) {
	f(t)
	for _, item := range t.items {
		if item != nil {
			item.Iterate(f)
		}
	}
}

func (t *TTuple) applyArgs(params map[ast.Identifier]Type, loc ast.Location) (Type, error) {
	var items []Type
	for _, item := range t.items {
		nItem, err := item.applyArgs(params, loc)
		if err != nil {
			return nil, err
		}
		items = append(items, nItem)

	}
	return NewTTuple(loc, items), nil
}

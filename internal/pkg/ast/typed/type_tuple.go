package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
)

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

func (t *TTuple) makeUnique(ctx *SolvingContext, ubMap map[uint64]uint64) Type {
	return NewTTuple(t.location, common.Map(func(x Type) Type { return x.makeUnique(ctx, ubMap) }, t.items))
}

func (t *TTuple) merge(other Type, loc ast.Location) (Equations, error) {
	if o, ok := other.(*TTuple); ok {
		if len(t.items) == len(o.items) {
			var eqs Equations
			for i, p := range t.items {
				eqs = append(eqs, NewEquationBestLoc(p, o.items[i], loc))
			}
			return eqs, nil
		}
	}
	return nil, newTypeMatchError(loc, t, other)
}

func (t *TTuple) mapTo(subst map[uint64]Type) (Type, error) {
	for i, p := range t.items {
		if x, err := p.mapTo(subst); err != nil {
			return nil, err
		} else {
			t.items[i] = x
		}
	}
	return t, nil
}

func (t *TTuple) equalsTo(other Type, req map[ast.FullIdentifier]struct{}) bool {
	ty, oky := other.(*TTuple)
	if oky {
		if len(t.items) != len(ty.items) {
			return false
		}
		for i, p := range t.items {
			if !p.equalsTo(ty.items[i], req) {
				return false
			}
		}
		return true
	}
	return false
}

func (t *TTuple) Children() []Statement {
	return common.Map(func(x Type) Statement { return x }, t.items)
}

func (t *TTuple) Code(currentModule ast.QualifiedIdentifier) string {
	return fmt.Sprintf("( %s )", common.Fold(func(x Type, s string) string {
		if s != "" {
			s += ", "
		}
		return s + x.Code("")
	}, "", t.items))
}

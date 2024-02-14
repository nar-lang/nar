package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
)

type TFunc struct {
	*typeBase
	params  []Type
	return_ Type
}

func NewTFunc(loc ast.Location, params []Type, return_ Type) Type {
	return &TFunc{
		typeBase: newTypeBase(loc),
		params:   params,
		return_:  return_,
	}
}

func (t *TFunc) makeUnique(ctx *SolvingContext, ubMap map[uint64]uint64) Type {
	return NewTFunc(t.location, common.Map(func(x Type) Type { return x.makeUnique(ctx, ubMap) }, t.params), t.return_.makeUnique(ctx, ubMap))
}

func (t *TFunc) merge(other Type, loc ast.Location) (Equations, error) {
	t1 := t
	if t2, ok := other.(*TFunc); ok {
		if len(t1.params) < len(t2.params) {
			t2 = t2.balance(len(t1.params))
		} else if len(t1.params) > len(t2.params) {
			t1 = t1.balance(len(t2.params))
		}
		var eqs Equations
		for i, p := range t1.params {
			eqs = append(eqs, NewEquationBestLoc(p, t2.params[i], loc))
		}
		eqs = append(eqs, NewEquationBestLoc(t1.return_, t2.return_, loc))
		return eqs, nil
	}
	return nil, newTypeMatchError(loc, t1, other)
}

func (t *TFunc) mapTo(subst map[uint64]Type) (Type, error) {
	for i, p := range t.params {
		if x, err := p.mapTo(subst); err != nil {
			return nil, err
		} else {
			t.params[i] = x
		}
	}
	var err error
	t.return_, err = t.return_.mapTo(subst)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (t *TFunc) equalsTo(other Type, req map[ast.FullIdentifier]struct{}) bool {
	ty, oky := other.(*TFunc)
	if oky {
		if len(t.params) != len(ty.params) {
			return false
		}
		for i, p := range t.params {
			if !p.equalsTo(ty.params[i], req) {
				return false
			}
		}
		return t.return_.equalsTo(ty.return_, req)
	}
	return false
}

func (t *TFunc) Children() []Statement {
	return append(common.Map(func(x Type) Statement { return x }, t.params), t.return_)
}

func (t *TFunc) Code(currentModule ast.QualifiedIdentifier) string {
	return fmt.Sprintf("(%s): %s", common.Fold(
		func(x Type, s string) string {
			if s != "" {
				s += ", "
			}
			return s + x.Code("")
		},
		"", t.params),
		t.return_.Code(""))
}

func (t *TFunc) NumParams() int {
	return len(t.params)
}

func (t *TFunc) ParamAt(index int) Type {
	return t.params[index]
}

func (t *TFunc) Return() Type {
	return t.return_
}

func (t *TFunc) balance(sz int) *TFunc {
	if len(t.params) == sz {
		return t
	}

	return NewTFunc(t.location, t.params[0:sz], NewTFunc(t.location, t.params[sz:], t.return_)).(*TFunc)
}

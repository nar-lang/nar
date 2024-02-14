package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
	"strings"
)

type TNative struct {
	*typeBase
	name ast.FullIdentifier
	args []Type
}

func NewTNative(loc ast.Location, name ast.FullIdentifier, args []Type) Type {
	return &TNative{
		typeBase: newTypeBase(loc),
		name:     name,
		args:     args,
	}
}

func (t *TNative) makeUnique(ctx *SolvingContext, ubMap map[uint64]uint64) Type {
	return NewTNative(t.location, t.name, common.Map(func(x Type) Type { return x.makeUnique(ctx, ubMap) }, t.args))
}

func (t *TNative) merge(other Type, loc ast.Location) (Equations, error) {
	if o, ok := other.(*TNative); ok {
		if o.name == t.name {
			if len(t.args) == len(o.args) {
				var eqs Equations
				for i, a := range t.args {
					eqs = append(eqs, NewEquationBestLoc(a, o.args[i], loc))
				}
				return eqs, nil
			}
		}
	}
	return nil, newTypeMatchError(loc, t, other)
}

func (t *TNative) mapTo(subst map[uint64]Type) (Type, error) {
	for i, a := range t.args {
		if x, err := a.mapTo(subst); err != nil {
			return nil, err
		} else {
			t.args[i] = x
		}
	}
	return t, nil
}

func (t *TNative) equalsTo(other Type, req map[ast.FullIdentifier]struct{}) bool {
	ty, oky := other.(*TNative)
	if oky {
		if t.name != ty.name {
			return false
		}
		if len(t.args) != len(ty.args) {
			return false
		}
		for i, a := range t.args {
			if !a.equalsTo(ty.args[i], req) {
				return false
			}
		}
		return true
	}
	return false
}

func (t *TNative) Children() []Statement {
	return common.Map(func(x Type) Statement { return x }, t.args)
}

func (t *TNative) Code(currentModule ast.QualifiedIdentifier) string {
	tp := common.Fold(func(x Type, s string) string {
		if s != "" {
			s += ", "
		}
		return s + x.Code("")
	}, "", t.args)
	if tp != "" {
		tp = "[" + tp + "]"
	}
	s := string(t.name)
	if currentModule != "" && strings.HasPrefix(s, string(currentModule)) {
		s = s[len(currentModule)+1:]
	}
	return fmt.Sprintf("%s%s", s, tp)
}

func (t *TNative) String() string {
	return string(t.name)
}

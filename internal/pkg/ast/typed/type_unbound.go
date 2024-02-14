package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
)

type TUnbound struct {
	*typeBase
	index       uint64
	constraint  common.Constraint
	givenName   ast.Identifier
	predecessor TypePredecessor
	solved      bool
}

func NewTParameter(ctx *SolvingContext, loc ast.Location, predecessor TypePredecessor, name ast.Identifier) Type {
	return ctx.annotateTypeParameter(loc, predecessor, name)
}

func newTUnbound(
	loc ast.Location, predecessor TypePredecessor, index uint64, constraint common.Constraint, givenName ast.Identifier,
) *TUnbound {
	return &TUnbound{
		typeBase:    newTypeBase(loc),
		index:       index,
		constraint:  constraint,
		givenName:   givenName,
		predecessor: predecessor,
	}
}

func (t *TUnbound) makeUnique(ctx *SolvingContext, ubMap map[uint64]uint64) Type {
	if x, ok := ubMap[t.index]; ok {
		return &TUnbound{
			typeBase:    newTypeBase(t.location),
			index:       x,
			constraint:  t.constraint,
			givenName:   t.givenName,
			predecessor: t.predecessor,
		}
	}
	ub := ctx.newAnnotatedConstraint(t, t.predecessor, t.givenName)
	ubMap[t.index] = ub.index
	return ub
}

func (t *TUnbound) merge(other Type, loc ast.Location) (Equations, error) {
	panic("should not be called")
}

func (t *TUnbound) mapTo(subst map[uint64]Type) (Type, error) {
	if t.solved {
		return t, nil
	}
	if x, ok := subst[t.index]; ok {
		if t.predecessor != nil {
			return t.predecessor.SetSuccessor(x), nil
		}
		return x.mapTo(subst)
	}
	return nil, common.Error{
		Location: t.location,
		Message:  fmt.Sprintf("failed to infer type"),
	}
}

func (t *TUnbound) equalsTo(other Type, req map[ast.FullIdentifier]struct{}) bool {
	ty, oky := other.(*TUnbound)
	if oky {
		return t.index == ty.index && t.constraint == ty.constraint
	}
	return false
}

func (t *TUnbound) Children() []Statement {
	return nil
}

func (t *TUnbound) Code(currentModule ast.QualifiedIdentifier) string {
	if t.givenName != "" {
		return string(t.givenName)
	}
	return fmt.Sprintf("u_%d%s", t.index, t.constraint)
}

func (t *TUnbound) String() string {
	return fmt.Sprintf("%d%s", t.index, t.constraint)
}

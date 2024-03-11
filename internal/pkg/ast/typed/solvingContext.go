package typed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
	"strings"
)

type annotationSource interface {
	ast.Coder
	Location() ast.Location
}

type SolvingContext struct {
	annotations    []annotationSource
	groups         []*typeGroup
	numSolvedTypes uint64
}

func newSolvingContext() *SolvingContext {
	return &SolvingContext{}
}

func (ctx *SolvingContext) annotateExpression(e Expression) Expression {
	e.setAnnotation(ctx.newTypeAnnotation(e))
	return e
}

func (ctx *SolvingContext) annotatePattern(p Pattern) Pattern {
	p.setAnnotation(ctx.newTypeAnnotation(p))
	return p
}

func (ctx *SolvingContext) annotateTypeParameter(loc ast.Location, predecessor TypePredecessor, name ast.Identifier) Type {
	t := ctx.newAnnotatedConstraint(&TUnbound{typeBase: &typeBase{location: loc}}, predecessor, name)
	ctx.annotations[t.index] = t
	return t
}

func (ctx *SolvingContext) newTypeAnnotation(stmt annotationSource) *TUnbound {
	return ctx.newAnnotatedConstraint(stmt, nil, "")
}

func (ctx *SolvingContext) newAnnotatedConstraint(stmt annotationSource, predecessor TypePredecessor, name ast.Identifier) *TUnbound {
	constraint := common.ConstraintNone
	if strings.HasPrefix(string(name), string(common.ConstraintNumber)) {
		constraint = common.ConstraintNumber
	}
	index := uint64(len(ctx.annotations))
	ctx.annotations = append(ctx.annotations, stmt)
	type_ := newTUnbound(stmt.Location(), predecessor, index, constraint, name)
	tg, _ := newTypeGroup(nil, type_, stmt.Location())
	ctx.groups = append(ctx.groups, tg)

	return type_
}

func (ctx *SolvingContext) newSolvedType(loc ast.Location, name ast.Identifier) Type {
	index := uint64(len(ctx.annotations)) + ctx.numSolvedTypes
	ctx.numSolvedTypes++
	t := newTUnbound(loc, nil, index, "", name)
	t.solved = true
	return t
}

func (ctx *SolvingContext) subst() map[uint64]Type {
	lastFreeName := 0
	subst := map[uint64]Type{}
	for _, tg := range ctx.groups {
		type_ := tg.specific
		if type_ == nil {
			if tg.givenName == "" {
				nameUsed := true
				var name ast.Identifier
				for nameUsed {
					nameUsed = false
					name = ast.Identifier('a' + rune(lastFreeName))
					for _, x := range ctx.groups {
						if x.givenName == name {
							nameUsed = true
							lastFreeName++
							break
						}
					}
				}
				tg.givenName = name
			}
			type_ = ctx.newSolvedType(tg.givenLoc, tg.givenName)
		}

		for ub := range tg.unbound {
			subst[ub] = type_
		}
	}
	return subst
}

type typeGroup struct {
	id         uint64
	specific   Type
	unbound    map[uint64]struct{}
	constraint common.Constraint
	givenName  ast.Identifier
	givenLoc   ast.Location
}

var lastGroupId = uint64(0)

func newTypeGroup(type_ Type, ub *TUnbound, loc ast.Location) (*typeGroup, error) {
	lastGroupId++

	tg := &typeGroup{
		id:      lastGroupId,
		unbound: map[uint64]struct{}{},
	}
	err := tg.absorb(ub, loc)
	if err != nil {
		return nil, err
	}
	if tub, ok := type_.(*TUnbound); ok {
		err := tg.absorb(tub, loc)
		return tg, err
	} else if type_ != nil {
		_, err := tg.specialize(type_, loc)
		return tg, err
	}
	return tg, nil
}

func (tg *typeGroup) containsUnbound(ub *TUnbound) bool {
	_, ok := tg.unbound[ub.index]
	return ok
}

func (tg *typeGroup) absorb(ub *TUnbound, loc ast.Location) error {
	if tg.constraint != "" && ub.constraint != "" && tg.constraint != ub.constraint {
		return common.NewErrorAt(loc, "type constraint violation")
	}
	if ub.constraint != "" {
		tg.constraint = ub.constraint
	}
	if tg.givenName == "" && ub.givenName != "" {
		tg.givenName = ub.givenName
		tg.givenLoc = ub.location
	}
	if ub.location.FilePath() == "" {
		tg.givenLoc = ub.location
	}
	tg.unbound[ub.index] = struct{}{}
	return nil
}

func (tg *typeGroup) merge(rg *typeGroup, loc ast.Location) (Equations, error) {
	for ub := range rg.unbound {
		tg.unbound[ub] = struct{}{}
	}

	if tg.constraint != "" && rg.constraint != "" && tg.constraint != rg.constraint {
		return nil, common.NewErrorAt(loc, "type constraint violation")
	}
	if rg.constraint != "" {
		tg.constraint = rg.constraint
	}
	if rg.specific != nil {
		return tg.specialize(rg.specific, loc)
	}
	return nil, nil
}

func (tg *typeGroup) specialize(type_ Type, loc ast.Location) (Equations, error) {
	switch tg.constraint {
	case common.ConstraintNumber:
		if n, ok := type_.(*TNative); !ok || (n.name != common.NarBaseMathInt && n.name != common.NarBaseMathFloat) {
			return nil, common.NewErrorAt(loc, "numeric type cannot hold %s", type_.Code(""))
		}
	}

	if tg.specific == nil {
		tg.specific = type_
		return nil, nil
	}
	return tg.specific.merge(type_, loc)
}

func (ctx *SolvingContext) insertAll(eqs Equations) (Equations, error) {
	for i := 0; i < len(eqs); i++ {
		eq := eqs[i]
		extra, err := ctx.insert(eq)
		eqs = appendUsefulEquations(eqs, extra)
		if err != nil {
			return eqs, err
		}
	}
	return eqs, nil
}

func appendUsefulEquations(eqs Equations, extra Equations) Equations {
	for _, eq := range extra {
		if eq.isRedundant() {
			continue
		}
		dup := false
		for _, eqx := range eqs {
			if eqx.equalsTo(eq) {
				dup = true
				break
			}
		}
		if dup {
			continue
		}
		eqs = append(eqs, eq)
	}
	return eqs
}

func (ctx *SolvingContext) insert(eq Equation) (Equations, error) {
	lUb, lIsUb := eq.left.(*TUnbound)
	rUb, rIsUb := eq.right.(*TUnbound)

	if lIsUb && rIsUb {
		return ctx.merge(lUb, rUb, eq.stmt.Location())
	} else if lIsUb {
		return ctx.specialize(lUb, eq.right, eq.stmt.Location())
	} else if rIsUb {
		return ctx.specialize(rUb, eq.left, eq.stmt.Location())
	} else {
		return eq.left.merge(eq.right, eq.stmt.Location())
	}
}

func (ctx *SolvingContext) specialize(ub *TUnbound, type_ Type, loc ast.Location) (Equations, error) {
	for _, tg := range ctx.groups {
		if tg.containsUnbound(ub) {
			return tg.specialize(type_, loc)
		}
	}
	return nil, common.NewErrorAt(ub.location, "cannot find annotation of `%s`", ub.Code(""))
}

func (ctx *SolvingContext) merge(l, r *TUnbound, loc ast.Location) (Equations, error) {
	var additional Equations
	for i := 0; i < len(ctx.groups); i++ {
		tga := ctx.groups[i]
		if tga.containsUnbound(l) {
			for j := len(ctx.groups) - 1; j >= 0; j-- {
				if i == j {
					continue
				}
				tgb := ctx.groups[j]
				if tgb.containsUnbound(r) {
					esq, err := tga.merge(tgb, loc)
					if err != nil {
						return nil, err
					}
					additional = append(additional, esq...)
					ctx.groups = append(ctx.groups[:j], ctx.groups[j+1:]...)
				}
			}
		}
	}
	return additional, nil
}

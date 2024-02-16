package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/bytecode"
	"nar-compiler/internal/pkg/common"
)

type PTuple struct {
	*patternBase
	items []Pattern
}

func NewPTuple(ctx *SolvingContext, loc ast.Location, declaredType Type, items []Pattern) Pattern {
	return ctx.annotatePattern(&PTuple{
		patternBase: newPatternBase(loc, declaredType),
		items:       items,
	})
}

func (p *PTuple) simplify() simplePattern {
	args := common.Map(func(x Pattern) simplePattern { return x.simplify() }, p.items)
	return simpleConstructor{
		Union: NewTData(
			p.location,
			ast.FullIdentifier(fmt.Sprintf("!!%d", len(p.items))),
			nil,
			[]*DataOption{
				NewDataOption("Only", common.Map(func(i Pattern) Type { return i.Type() }, p.items)),
			},
		).(*TData),
		Name: "Only",
		Args: args,
	}
}

func (p *PTuple) mapTypes(subst map[uint64]Type) error {
	var err error
	p.type_, err = p.type_.mapTo(subst)
	if err != nil {
		return err
	}
	for _, item := range p.items {
		err = item.mapTypes(subst)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *PTuple) Code(currentModule ast.QualifiedIdentifier) string {
	s := "(" + common.Fold(func(x Pattern, s string) string {
		if s != "" {
			s += ", "
		}
		return s + x.Code(currentModule)
	}, "", p.items) + ")"
	return s
}

func (p *PTuple) Children() []Statement {
	return append(p.patternBase.Children(), common.Map(func(x Pattern) Statement { return x }, p.items)...)
}

func (p *PTuple) appendBytecode(ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary) ([]bytecode.Op, []ast.Location) {
	var err error
	for _, item := range p.items {
		ops, locations = item.appendBytecode(ops, locations, binary)
		if err != nil {
			return nil, nil
		}
	}
	return bytecode.AppendMakePattern(bytecode.PatternKindTuple, "", len(p.items), p.location, ops, locations, binary)
}

func (p *PTuple) appendEquations(eqs Equations, loc *ast.Location, localDefs localTypesMap, ctx *SolvingContext, stack []*Definition) (Equations, error) {
	var err error
	items, err := common.MapError(func(e Pattern) (Type, error) {
		t := e.Type()
		if t == nil {
			return nil, common.NewErrorOf(e, "type cannot be inferred")
		}
		return t, nil
	}, p.items)
	if err != nil {
		return nil, err
	}

	eqs = append(eqs, NewEquation(p, p.type_, NewTTuple(p.location, items)))

	for _, item := range p.items {
		eqs, err = item.appendEquations(eqs, loc, localDefs, ctx, stack)
		if err != nil {
			return nil, err
		}
	}

	if p.declaredType != nil {
		eqs = append(eqs, NewEquation(p, p.type_, p.declaredType))
	}

	return eqs, nil

}

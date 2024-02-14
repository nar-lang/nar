package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/bytecode"
	"nar-compiler/internal/pkg/common"
)

type PList struct {
	*patternBase
	items    []Pattern
	itemType Type
	ctx      *SolvingContext
}

func NewPList(ctx *SolvingContext, loc ast.Location, declaredType Type, items []Pattern) Pattern {
	p := &PList{
		patternBase: newPatternBase(loc, declaredType),
		items:       items,
		ctx:         ctx,
	}
	p.itemType = ctx.newTypeAnnotation(p)
	return ctx.annotatePattern(p)
}

func (p *PList) simplify() simplePattern {
	var nested []simplePattern
	ctor := "Nil"
	if len(p.items) > 0 {
		item := NewPList(p.ctx, p.location, nil, p.items[1:]).simplify()
		ctor = "Cons"
		nested = []simplePattern{item}
	}
	a := p.ctx.newTypeAnnotation(p)
	return simpleConstructor{
		Union: NewTData(p.location, "!!list", nil, []*DataOption{
			NewDataOption("Nil", nil),
			NewDataOption("Cons", []Type{a, NewTNative(p.location, common.NarBaseListList, []Type{a})}),
		}).(*TData),
		Name: ast.DataOptionIdentifier(ctor),
		Args: nested,
	}
}

func (p *PList) mapTypes(subst map[uint64]Type) error {
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

func (p *PList) Code(currentModule ast.QualifiedIdentifier) string {
	s := fmt.Sprintf("[%s]",
		common.Fold(func(x Pattern, s string) string {
			if s != "" {
				s += ", "
			}
			return s + x.Code(currentModule)
		}, "", p.items))
	if p.declaredType != nil {
		s += ": " + p.declaredType.Code(currentModule)
	}
	return s
}

func (p *PList) Children() []Statement {
	return append(p.patternBase.Children(), common.Map(func(x Pattern) Statement { return x }, p.items)...)
}

func (p *PList) appendBytecode(ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary) ([]bytecode.Op, []ast.Location) {
	var err error
	for _, item := range p.items {
		ops, locations = item.appendBytecode(ops, locations, binary)
		if err != nil {
			return nil, nil
		}
	}
	return bytecode.AppendMakePattern(bytecode.PatternKindList, "", len(p.items), p.location, ops, locations, binary)
}

func (p *PList) appendEquations(eqs Equations, loc *ast.Location, localDefs localTypesMap, ctx *SolvingContext, stack []*Definition) (Equations, error) {
	var err error
	for _, item := range p.items {
		eqs = append(eqs, NewEquation(item, p.itemType, item.Type()))
	}
	typeNative := NewTNative(p.location, common.NarBaseListList, []Type{p.itemType})
	eqs = append(eqs, NewEquation(p, p.type_, typeNative))

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

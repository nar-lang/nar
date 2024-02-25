package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
	bytecode "nar-compiler/pkg/bytecode"
)

type PCons struct {
	*patternBase
	head, tail Pattern
	ctx        *SolvingContext
}

func NewPCons(ctx *SolvingContext, loc ast.Location, declaredType Type, head Pattern, tail Pattern) Pattern {
	return ctx.annotatePattern(&PCons{
		patternBase: newPatternBase(loc, declaredType),
		head:        head,
		tail:        tail,
		ctx:         ctx,
	})
}

func (p *PCons) simplify() simplePattern {
	a := p.ctx.newTypeAnnotation(p)
	head := p.head.simplify()
	tail := p.tail.simplify()
	return simpleConstructor{
		Union: NewTData(p.location, "!!list", nil, []*DataOption{
			NewDataOption("Nil", nil),
			NewDataOption("Cons", []Type{a, NewTNative(p.location, common.NarBaseListList, []Type{a})}),
		}).(*TData),
		Name: "Cons",
		Args: []simplePattern{head, tail},
	}
}

func (p *PCons) mapTypes(subst map[uint64]Type) error {
	var err error
	p.type_, err = p.type_.mapTo(subst)
	if err != nil {
		return err
	}
	err = p.head.mapTypes(subst)
	if err != nil {
		return err
	}
	return p.tail.mapTypes(subst)
}

func (p *PCons) Code(currentModule ast.QualifiedIdentifier) string {
	s := fmt.Sprintf("(%s | %s)", p.head.Code(currentModule), p.tail.Code(currentModule))
	if p.declaredType != nil {
		s += ": " + p.declaredType.Code(currentModule)
	}
	return s
}

func (p *PCons) Children() []Statement {
	return append(p.patternBase.Children(), p.head, p.tail)
}

func (p *PCons) appendBytecode(ops []bytecode.Op, locations []bytecode.Location, binary *bytecode.Binary, hash *bytecode.BinaryHash) ([]bytecode.Op, []bytecode.Location) {
	var err error
	ops, locations = p.tail.appendBytecode(ops, locations, binary, hash)
	if err != nil {
		return nil, nil
	}
	ops, locations = p.head.appendBytecode(ops, locations, binary, hash)
	if err != nil {
		return nil, nil
	}
	return bytecode.AppendMakePattern(bytecode.PatternKindCons, "", 0, p.location.Bytecode(), ops, locations, binary, hash)
}

func (p *PCons) appendEquations(eqs Equations, loc *ast.Location, localDefs localTypesMap, ctx *SolvingContext, stack []*Definition) (Equations, error) {
	var err error
	typeNative := NewTNative(p.location, common.NarBaseListList, []Type{p.head.Type()})
	eqs, err = p.head.appendEquations(eqs, loc, localDefs, ctx, stack)
	if err != nil {
		return nil, err
	}
	eqs, err = p.tail.appendEquations(eqs, loc, localDefs, ctx, stack)
	if err != nil {
		return nil, err
	}
	eqs = append(eqs,
		NewEquation(p, p.type_, p.tail.Type()),
		NewEquation(p, p.tail.Type(), typeNative))

	if p.declaredType != nil {
		eqs = append(eqs, NewEquation(p, p.type_, p.declaredType))
	}
	return eqs, nil
}

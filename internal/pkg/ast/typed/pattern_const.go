package typed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/bytecode"
)

type PConst struct {
	*patternBase
	value ast.ConstValue
}

func NewPConst(ctx *SolvingContext, loc ast.Location, declaredType Type, value ast.ConstValue) Pattern {
	return ctx.annotatePattern(&PConst{
		patternBase: newPatternBase(loc, declaredType),
		value:       value,
	})
}

func (p *PConst) simplify() simplePattern {
	if _, ok := p.value.(ast.CUnit); ok {
		return simpleConstructor{
			Union: NewTData(p.location, "!!Unit", nil, []*DataOption{NewDataOption("Only", nil)}).(*TData),
			Name:  "Only",
		}
	}
	return simpleLiteral{p.value}
}

func (p *PConst) mapTypes(subst map[uint64]Type) error {
	var err error
	p.type_, err = p.type_.mapTo(subst)
	return err
}

func (p *PConst) Code(currentModule ast.QualifiedIdentifier) string {
	s := p.value.Code(currentModule)
	if p.declaredType != nil {
		s += ": " + p.declaredType.Code(currentModule)
	}
	return s
}

func (p *PConst) appendBytecode(ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary) ([]bytecode.Op, []ast.Location) {
	ops, locations = bytecode.AppendLoadConstValue(p.value, bytecode.StackKindPattern, p.location, ops, locations, binary)
	return bytecode.AppendMakePattern(bytecode.PatternKindConst, "", 0, p.location, ops, locations, binary)
}

func (p *PConst) appendEquations(eqs Equations, loc *ast.Location, localDefs localTypesMap, ctx *SolvingContext, stack []*Definition) (Equations, error) {
	if p.declaredType != nil {
		eqs = append(eqs, NewEquation(p, p.type_, p.declaredType))
	}
	return eqs, nil
}
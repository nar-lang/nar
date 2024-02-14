package typed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/bytecode"
)

type PAny struct {
	*patternBase
}

func NewPAny(ctx *SolvingContext, loc ast.Location, declaredType Type) Pattern {
	return ctx.annotatePattern(&PAny{
		patternBase: newPatternBase(loc, declaredType),
	})
}

func (p *PAny) simplify() simplePattern {
	return simpleAnything{}
}

func (p *PAny) mapTypes(subst map[uint64]Type) error {
	var err error
	p.type_, err = p.type_.mapTo(subst)
	return err
}

func (p *PAny) Code(currentModule ast.QualifiedIdentifier) string {
	s := "_"
	if p.declaredType != nil {
		s += ": " + p.declaredType.Code(currentModule)
	}
	return s
}

func (p *PAny) appendBytecode(ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary) ([]bytecode.Op, []ast.Location) {
	return bytecode.AppendMakePattern(bytecode.PatternKindAny, "", 0, p.location, ops, locations, binary)
}

func (p *PAny) appendEquations(eqs Equations, loc *ast.Location, localDefs localTypesMap, ctx *SolvingContext, stack []*Definition) (Equations, error) {
	if p.declaredType != nil {
		eqs = append(eqs, NewEquation(p, p.type_, p.declaredType))
	}
	return eqs, nil
}

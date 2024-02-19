package typed

import (
	"nar-compiler/internal/pkg/ast"
)

type Pattern interface {
	Statement
	bytecoder
	_pattern()
	Type() Type
	DeclaredType() Type
	appendEquations(eqs Equations, loc *ast.Location, localDefs localTypesMap, ctx *SolvingContext, stack []*Definition) (Equations, error)
	mapTypes(subst map[uint64]Type) error
	simplify() simplePattern
	setAnnotation(annotation *TUnbound)
}

type patternBase struct {
	location     ast.Location
	type_        Type
	declaredType Type
}

func newPatternBase(loc ast.Location, declaredType Type) *patternBase {
	return &patternBase{
		location:     loc,
		declaredType: declaredType,
	}
}

func (p *patternBase) _pattern() {}

func (p *patternBase) Location() ast.Location {
	return p.location
}

func (p *patternBase) Type() Type {
	return p.type_
}

func (p *patternBase) DeclaredType() Type {
	return p.declaredType
}

func (p *patternBase) Children() []Statement {
	return []Statement{p.type_}
}

func (p *patternBase) setAnnotation(annotation *TUnbound) {
	p.type_ = annotation
}

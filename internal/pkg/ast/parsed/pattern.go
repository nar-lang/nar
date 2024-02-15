package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

type Pattern interface {
	Statement
	normalize(
		locals map[ast.Identifier]normalized.Pattern,
		modules map[ast.QualifiedIdentifier]*Module,
		module *Module,
		normalizedModule *normalized.Module,
	) (normalized.Pattern, error)
	Type() Type
	SetDeclaredType(decl Type)
	setSuccessor(n normalized.Pattern) normalized.Pattern
}

func newPatternBase(loc ast.Location) *patternBase {
	return &patternBase{
		location: loc,
	}
}

type patternBase struct {
	location     ast.Location
	declaredType Type
	successor    normalized.Pattern
}

func (p *patternBase) Location() ast.Location {
	return p.location
}

func (*patternBase) _parsed() {}

func (p *patternBase) SetDeclaredType(t Type) {
	p.declaredType = t
}

func (p *patternBase) Type() Type {
	return p.declaredType
}

func (p *patternBase) Successor() normalized.Statement {
	return p.successor
}

func (p *patternBase) setSuccessor(n normalized.Pattern) normalized.Pattern {
	p.successor = n
	return n
}

func (p *patternBase) Iterate(f func(statement Statement)) {
	if p.declaredType != nil {
		f(p.declaredType)
	}
}

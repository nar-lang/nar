package typed

import (
	"nar-compiler/internal/pkg/ast"
	bytecode "nar-compiler/pkg/bytecode"
)

type PNamed struct {
	*patternBase
	name ast.Identifier
}

func NewPNamed(ctx *SolvingContext, loc ast.Location, declaredType Type, name ast.Identifier) Pattern {
	return ctx.annotatePattern(&PNamed{
		patternBase: newPatternBase(loc, declaredType),
		name:        name,
	})
}

func (p *PNamed) simplify() simplePattern {
	return simpleAnything{}
}

func (p *PNamed) mapTypes(subst map[uint64]Type) error {
	var err error
	p.type_, err = p.type_.mapTo(subst)
	return err
}

func (p *PNamed) Code(currentModule ast.QualifiedIdentifier) string {
	s := string(p.name)
	if p.declaredType != nil {
		s += ": " + p.declaredType.Code(currentModule)
	}
	return s
}

func (p *PNamed) appendBytecode(ops []bytecode.Op, locations []bytecode.Location, binary *bytecode.Binary) ([]bytecode.Op, []bytecode.Location) {
	return bytecode.AppendMakePattern(bytecode.PatternKindNamed, string(p.name), 0, p.location.Bytecode(), ops, locations, binary)
}

func (p *PNamed) appendEquations(eqs Equations, loc *ast.Location, localDefs localTypesMap, ctx *SolvingContext, stack []*Definition) (Equations, error) {
	localDefs[p.name] = p.type_
	if p.declaredType != nil {
		eqs = append(eqs, NewEquation(p, p.type_, p.declaredType))
	}
	return eqs, nil
}

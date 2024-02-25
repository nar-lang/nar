package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	bytecode "nar-compiler/pkg/bytecode"
)

type PAlias struct {
	*patternBase
	alias  ast.Identifier
	nested Pattern
}

func (p *PAlias) simplify() simplePattern {
	return p.nested.simplify()
}

func NewPAlias(ctx *SolvingContext, loc ast.Location, declaredType Type, alias ast.Identifier, nested Pattern) Pattern {
	return ctx.annotatePattern(&PAlias{
		patternBase: newPatternBase(loc, declaredType),
		alias:       alias,
		nested:      nested,
	})
}

func (p *PAlias) mapTypes(subst map[uint64]Type) error {
	var err error
	p.type_, err = p.type_.mapTo(subst)
	if err != nil {
		return err
	}
	return p.nested.mapTypes(subst)
}

func (p *PAlias) Children() []Statement {
	return append(p.patternBase.Children(), p.nested)
}

func (p *PAlias) Code(currentModule ast.QualifiedIdentifier) string {
	s := fmt.Sprintf("(%s as %s)", p.nested.Code(currentModule), p.alias)
	if p.declaredType != nil {
		s += ": " + p.declaredType.Code(currentModule)
	}
	return s
}

func (p *PAlias) appendBytecode(ops []bytecode.Op, locations []bytecode.Location, binary *bytecode.Binary, hash *bytecode.BinaryHash) ([]bytecode.Op, []bytecode.Location) {
	var err error
	ops, locations = p.nested.appendBytecode(ops, locations, binary, hash)
	if err != nil {
		return nil, nil
	}
	return bytecode.AppendMakePattern(bytecode.PatternKindAlias, string(p.alias), 0, p.location.Bytecode(), ops, locations, binary, hash)
}

func (p *PAlias) appendEquations(eqs Equations, loc *ast.Location, localDefs localTypesMap, ctx *SolvingContext, stack []*Definition) (Equations, error) {
	localDefs[p.alias] = p.type_

	var err error
	eqs, err = p.nested.appendEquations(eqs, loc, localDefs, ctx, stack)
	if err != nil {
		return nil, err
	}
	eqs = append(eqs, NewEquation(p, p.type_, p.nested.Type()))

	if p.declaredType != nil {
		eqs = append(eqs, NewEquation(p, p.type_, p.declaredType))
	}
	return eqs, nil
}

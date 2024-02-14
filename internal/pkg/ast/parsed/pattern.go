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
	SetType(decl Type)
	Successor() normalized.Pattern
	setSuccessor(n normalized.Pattern)
}

type patternBase struct {
	location  ast.Location
	type_     Type
	successor normalized.Pattern
}

func newPatternBase(loc ast.Location) *patternBase {
	return &patternBase{
		location: loc,
	}
}

func (p *patternBase) GetLocation() ast.Location {
	return p.location
}

func (*patternBase) _parsed() {}

func (p *patternBase) SetType(t Type) {
	p.type_ = t
}

func (p *patternBase) Type() Type {
	return p.type_
}

func (p *patternBase) Successor() normalized.Pattern {
	return p.successor
}

func (p *patternBase) setSuccessor(n normalized.Pattern) {
	p.successor = n
}

type PAlias struct {
	*patternBase
	alias  ast.Identifier
	nested Pattern
}

func NewPAlias(loc ast.Location, alias ast.Identifier, nested Pattern) Pattern {
	return &PAlias{
		patternBase: newPatternBase(loc),
		alias:       alias,
		nested:      nested,
	}
}

type PAny struct {
	*patternBase
}

func NewPAny(loc ast.Location) Pattern {
	return &PAny{
		patternBase: newPatternBase(loc),
	}
}

type PCons struct {
	*patternBase
	head, tail Pattern
}

func NewPCons(loc ast.Location, head, tail Pattern) Pattern {
	return &PCons{
		patternBase: newPatternBase(loc),
		head:        head,
		tail:        tail,
	}
}

type PConst struct {
	*patternBase
	value ast.ConstValue
}

func NewPConst(loc ast.Location, value ast.ConstValue) Pattern {
	return &PConst{
		patternBase: newPatternBase(loc),
		value:       value,
	}
}

type PDataOption struct {
	*patternBase
	name   ast.QualifiedIdentifier
	values []Pattern
}

func NewPDataOption(loc ast.Location, name ast.QualifiedIdentifier, values []Pattern) Pattern {
	return &PDataOption{
		patternBase: newPatternBase(loc),
		name:        name,
		values:      values,
	}
}

type PList struct {
	*patternBase
	items []Pattern
}

func NewPList(loc ast.Location, items []Pattern) Pattern {
	return &PList{
		patternBase: newPatternBase(loc),
		items:       items,
	}
}

type PNamed struct {
	*patternBase
	name ast.Identifier
}

func NewPNamed(loc ast.Location, name ast.Identifier) Pattern {
	return &PNamed{
		patternBase: newPatternBase(loc),
		name:        name,
	}
}

func (p *PNamed) Name() ast.Identifier {
	return p.name
}

type PRecordField struct {
	location ast.Location
	name     ast.Identifier
}

func NewPRecordField(loc ast.Location, name ast.Identifier) PRecordField {
	return PRecordField{
		location: loc,
		name:     name,
	}
}

type PRecord struct {
	*patternBase
	fields []PRecordField
}

func NewPRecord(loc ast.Location, fields []PRecordField) Pattern {
	return &PRecord{
		patternBase: newPatternBase(loc),
		fields:      fields,
	}
}

type PTuple struct {
	*patternBase
	items []Pattern
}

func NewPTuple(loc ast.Location, items []Pattern) Pattern {
	return &PTuple{
		patternBase: newPatternBase(loc),
		items:       items,
	}
}

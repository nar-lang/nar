package parsed

import (
	"nar-compiler/internal/pkg/ast"
)

type Pattern interface {
	_pattern()
	WithType(decl Type) Pattern
	GetLocation() ast.Location
	GetType() Type
}

type PAlias struct {
	ast.Location
	Type
	Alias  ast.Identifier
	Nested Pattern
}

func (PAlias) _pattern() {}

func (p PAlias) WithType(decl Type) Pattern {
	p.Type = decl
	return p
}

func (p PAlias) GetLocation() ast.Location {
	return p.Location
}

func (p PAlias) GetType() Type {
	return p.Type
}

type PAny struct {
	ast.Location
	Type
}

func (PAny) _pattern() {}

func (p PAny) WithType(decl Type) Pattern {
	p.Type = decl
	return p
}

func (p PAny) GetLocation() ast.Location {
	return p.Location
}

func (p PAny) GetType() Type {
	return p.Type
}

type PCons struct {
	ast.Location
	Type
	Head, Tail Pattern
}

func (PCons) _pattern() {}

func (p PCons) WithType(decl Type) Pattern {
	p.Type = decl
	return p
}

func (p PCons) GetLocation() ast.Location {
	return p.Location
}

func (p PCons) GetType() Type {
	return p.Type
}

type PConst struct {
	ast.Location
	Type
	Value ast.ConstValue
}

func (PConst) _pattern() {}

func (p PConst) WithType(decl Type) Pattern {
	p.Type = decl
	return p
}

func (p PConst) GetLocation() ast.Location {
	return p.Location
}

func (p PConst) GetType() Type {
	return p.Type
}

type PDataOption struct {
	ast.Location
	Type
	Name   ast.QualifiedIdentifier
	Values []Pattern
}

func (PDataOption) _pattern() {}

func (p PDataOption) WithType(decl Type) Pattern {
	p.Type = decl
	return p
}

func (p PDataOption) GetLocation() ast.Location {
	return p.Location
}

func (p PDataOption) GetType() Type {
	return p.Type
}

type PList struct {
	ast.Location
	Type
	Items []Pattern
}

func (PList) _pattern() {}

func (p PList) WithType(decl Type) Pattern {
	p.Type = decl
	return p
}

func (p PList) GetLocation() ast.Location {
	return p.Location
}

func (p PList) GetType() Type {
	return p.Type
}

type PNamed struct {
	ast.Location
	Type
	Name ast.Identifier
}

func (PNamed) _pattern() {}

func (p PNamed) WithType(decl Type) Pattern {
	p.Type = decl
	return p
}

func (p PNamed) GetLocation() ast.Location {
	return p.Location
}

func (p PNamed) GetType() Type {
	return p.Type
}

type PRecordField struct {
	ast.Location
	Name ast.Identifier
}

type PRecord struct {
	ast.Location
	Type
	Fields []PRecordField
}

func (PRecord) _pattern() {}

func (p PRecord) WithType(decl Type) Pattern {
	p.Type = decl
	return p
}

func (p PRecord) GetLocation() ast.Location {
	return p.Location
}

func (p PRecord) GetType() Type {
	return p.Type
}

type PTuple struct {
	ast.Location
	Type
	Items []Pattern
}

func (PTuple) _pattern() {}

func (p PTuple) WithType(decl Type) Pattern {
	p.Type = decl
	return p
}

func (p PTuple) GetLocation() ast.Location {
	return p.Location
}

func (p PTuple) GetType() Type {
	return p.Type
}

package normalized

import (
	"nar-compiler/internal/pkg/ast"
)

type Pattern interface {
	_pattern()
}

type PAlias struct {
	ast.Location
	Type
	Alias  ast.Identifier
	Nested Pattern
}

func (PAlias) _pattern() {}

type PAny struct {
	ast.Location
	Type
}

func (PAny) _pattern() {}

type PCons struct {
	ast.Location
	Type
	Head, Tail Pattern
}

func (PCons) _pattern() {}

type PConst struct {
	ast.Location
	Type
	Value ast.ConstValue
}

func (PConst) _pattern() {}

type PDataOption struct {
	ast.Location
	Type
	ModuleName     ast.QualifiedIdentifier
	DefinitionName ast.Identifier
	Values         []Pattern
}

func (PDataOption) _pattern() {}

type PList struct {
	ast.Location
	Type
	Items []Pattern
}

func (PList) _pattern() {}

type PNamed struct {
	ast.Location
	Type
	Name ast.Identifier
}

func (PNamed) _pattern() {}

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

type PTuple struct {
	ast.Location
	Type
	Items []Pattern
}

func (PTuple) _pattern() {}

type PUnit struct {
	ast.Location
	Type
}

func (PUnit) _pattern() {}

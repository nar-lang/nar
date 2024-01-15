package normalized

import (
	"nar-compiler/internal/pkg/ast"
)

type Pattern interface {
	_pattern()
	GetLocation() ast.Location
}

type PatternBase struct {
	Location ast.Location
}

func (p *PatternBase) _pattern() {}

func (p *PatternBase) GetLocation() ast.Location {
	return p.Location
}

type PAlias struct {
	*PatternBase
	Type   Type
	Alias  ast.Identifier
	Nested Pattern
}

type PAny struct {
	*PatternBase
	Type Type
}

type PCons struct {
	*PatternBase
	Type       Type
	Head, Tail Pattern
}

type PConst struct {
	*PatternBase
	Type  Type
	Value ast.ConstValue
}

type PDataOption struct {
	*PatternBase
	Type           Type
	ModuleName     ast.QualifiedIdentifier
	DefinitionName ast.Identifier
	Values         []Pattern
}

type PList struct {
	*PatternBase
	Type  Type
	Items []Pattern
}

type PNamed struct {
	*PatternBase
	Type Type
	Name ast.Identifier
}

type PRecordField struct {
	Location ast.Location
	Name     ast.Identifier
}

type PRecord struct {
	*PatternBase
	Type   Type
	Fields []PRecordField
}

type PTuple struct {
	*PatternBase
	Type  Type
	Items []Pattern
}

type PUnit struct {
	*PatternBase
	Type Type
}

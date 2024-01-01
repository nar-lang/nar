package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
)

type Pattern interface {
	fmt.Stringer
	_pattern()
	GetType() Type
	GetLocation() ast.Location
}

type PAlias struct {
	ast.Location
	Type
	Alias  ast.Identifier
	Nested Pattern
}

func (*PAlias) _pattern() {}

func (p *PAlias) String() string {
	return fmt.Sprintf("PAlias(%s,%v){%s}", p.Alias, p.Nested, p.Type)
}

func (p *PAlias) GetLocation() ast.Location {
	return p.Location
}

func (p *PAlias) GetType() Type {
	return p.Type
}

type PAny struct {
	ast.Location
	Type
}

func (*PAny) _pattern() {}

func (p *PAny) String() string {
	return fmt.Sprintf("PAny(){%s}", p.Type)
}

func (p *PAny) GetLocation() ast.Location {
	return p.Location
}

func (p *PAny) GetType() Type {
	return p.Type
}

type PCons struct {
	ast.Location
	Type
	Head, Tail Pattern
}

func (*PCons) _pattern() {}

func (p *PCons) String() string {
	return fmt.Sprintf("PCons(%v,%v){%s}", p.Head, p.Tail, p.Type)
}

func (p *PCons) GetLocation() ast.Location {
	return p.Location
}

func (p *PCons) GetType() Type {
	return p.Type
}

type PConst struct {
	ast.Location
	Type
	Value ast.ConstValue
}

func (*PConst) _pattern() {}

func (p *PConst) String() string {
	return fmt.Sprintf("PConst(%v){%s}", p.Value, p.Type)
}

func (p *PConst) GetLocation() ast.Location {
	return p.Location
}

func (p *PConst) GetType() Type {
	return p.Type
}

type PDataOption struct {
	ast.Location
	Type
	DataName   ast.FullIdentifier
	OptionName ast.Identifier
	Definition *Definition
	Args       []Pattern
}

func (*PDataOption) _pattern() {}

func (p *PDataOption) String() string {
	return fmt.Sprintf(
		"PDataOption(%s,%v){%s}",
		common.MakeDataOptionIdentifier(p.DataName, p.OptionName),
		p.Args, p.Type)
}

func (p *PDataOption) GetLocation() ast.Location {
	return p.Location
}

func (p *PDataOption) GetType() Type {
	return p.Type
}

type PList struct {
	ast.Location
	Type
	Items []Pattern
}

func (*PList) _pattern() {}

func (p *PList) String() string {
	return fmt.Sprintf("PList(%v){%s}", p.Items, p.Type)
}

func (p *PList) GetLocation() ast.Location {
	return p.Location
}

func (p *PList) GetType() Type {
	return p.Type
}

type PNamed struct {
	ast.Location
	Type
	Name ast.Identifier
}

func (*PNamed) _pattern() {}

func (p *PNamed) String() string {
	return fmt.Sprintf("PNamed(%s){%s}", p.Name, p.Type)
}

func (p *PNamed) GetLocation() ast.Location {
	return p.Location
}

func (p *PNamed) GetType() Type {
	return p.Type
}

type PRecordField struct {
	ast.Location
	Name ast.Identifier
	Type Type
}

type PRecord struct {
	ast.Location
	Type
	Fields []PRecordField
}

func (*PRecord) _pattern() {}

func (p *PRecord) String() string {
	return fmt.Sprintf("PRecord(%+v){%s}", p.Fields, p.Type)
}

func (p *PRecord) GetLocation() ast.Location {
	return p.Location
}

func (p *PRecord) GetType() Type {
	return p.Type
}

type PTuple struct {
	ast.Location
	Type
	Items []Pattern
}

func (*PTuple) _pattern() {}

func (p *PTuple) String() string {
	return fmt.Sprintf("PTuple(%v){%s}", p.Items, p.Type)
}

func (p *PTuple) GetLocation() ast.Location {
	return p.Location
}

func (p *PTuple) GetType() Type {
	return p.Type
}

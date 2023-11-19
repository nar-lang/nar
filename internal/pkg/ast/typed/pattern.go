package typed

import (
	"fmt"
	"oak-compiler/internal/pkg/ast"
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
	return fmt.Sprintf("PAlias(%s,%v)", p.Alias, p.Nested)
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
	return fmt.Sprintf("PAny()")
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
	return fmt.Sprintf("PCons(%v,%v)", p.Head, p.Tail)
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
	return fmt.Sprintf("PConst(%v)", p.Value)
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
	Name       ast.DataOptionIdentifier
	Definition *Definition
	Args       []Pattern
}

func (*PDataOption) _pattern() {}

func (p *PDataOption) String() string {
	return fmt.Sprintf("PDataOption(%s,%v)", p.Name, p.Args)
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
	return fmt.Sprintf("PList(%v)", p.Items)
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
	return fmt.Sprintf("PNamed(%s)", p.Name)
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
	return fmt.Sprintf("PRecord(%+v)", p.Fields)
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
	return fmt.Sprintf("PTuple(%v)", p.Items)
}

func (p *PTuple) GetLocation() ast.Location {
	return p.Location
}

func (p *PTuple) GetType() Type {
	return p.Type
}

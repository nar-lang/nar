package normalized

import (
	"nar-compiler/internal/pkg/ast"
)

type Type interface {
	_type()
	GetLocation() ast.Location
}

type TFunc struct {
	ast.Location
	Params []Type
	Return Type
}

func (*TFunc) _type() {}

func (t *TFunc) GetLocation() ast.Location {
	return t.Location
}

type TRecord struct {
	ast.Location
	Fields map[ast.Identifier]Type
}

func (*TRecord) _type() {}

func (t *TRecord) GetLocation() ast.Location {
	return t.Location
}

type TTuple struct {
	ast.Location
	Items []Type
}

func (*TTuple) _type() {}

func (t *TTuple) GetLocation() ast.Location {
	return t.Location
}

type TUnit struct {
	ast.Location
}

func (*TUnit) _type() {}

func (t *TUnit) GetLocation() ast.Location {
	return t.Location
}

type DataOption struct {
	Name   ast.Identifier
	Hidden bool
	Values []Type
}

type TData struct {
	ast.Location
	Name    ast.FullIdentifier
	Args    []Type
	Options []DataOption
}

func (*TData) _type() {}

func (t *TData) GetLocation() ast.Location {
	return t.Location
}

type TNative struct {
	ast.Location
	Name ast.FullIdentifier
	Args []Type
}

func (*TNative) _type() {}

func (t *TNative) GetLocation() ast.Location {
	return t.Location
}

type TTypeParameter struct {
	ast.Location
	Name ast.Identifier
}

func (*TTypeParameter) _type() {}

func (t *TTypeParameter) GetLocation() ast.Location {
	return t.Location
}

func (p *TTypeParameter) String() string {
	return string(p.Name)
}

type TPlaceholder struct {
	Name ast.FullIdentifier
}

func (p *TPlaceholder) _type() {}

func (p *TPlaceholder) GetLocation() ast.Location {
	return ast.Location{}
}

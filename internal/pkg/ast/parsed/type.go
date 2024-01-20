package parsed

import (
	"nar-compiler/internal/pkg/ast"
)

type Type interface {
	_type()
	GetLocation() ast.Location
}

type TFunc struct {
	Location ast.Location
	Params   []Type
	Return   Type
}

func (*TFunc) _type() {}

func (t *TFunc) GetLocation() ast.Location {
	return t.Location
}

type TRecord struct {
	Location ast.Location
	Fields   map[ast.Identifier]Type
}

func (*TRecord) _type() {}

func (t *TRecord) GetLocation() ast.Location {
	return t.Location
}

type TTuple struct {
	Location ast.Location
	Items    []Type
}

func (*TTuple) _type() {}

func (t *TTuple) GetLocation() ast.Location {
	return t.Location
}

type TUnit struct {
	Location ast.Location
}

func (*TUnit) _type() {}

func (t *TUnit) GetLocation() ast.Location {
	return t.Location
}

type TNamed struct {
	Location ast.Location
	Name     ast.QualifiedIdentifier
	Args     []Type
}

func (*TNamed) _type() {}

func (t *TNamed) GetLocation() ast.Location {
	return t.Location
}

type DataOption struct {
	Name   ast.Identifier
	Hidden bool
	Values []Type
}

type TData struct {
	Location ast.Location
	Name     ast.FullIdentifier
	Args     []Type
	Options  []DataOption
}

func (*TData) _type() {}

func (t *TData) GetLocation() ast.Location {
	return t.Location
}

type TNative struct {
	Location ast.Location
	Name     ast.FullIdentifier
	Args     []Type
}

func (*TNative) _type() {}

func (t *TNative) GetLocation() ast.Location {
	return t.Location
}

type TTypeParameter struct {
	Location ast.Location
	Name     ast.Identifier
}

func (*TTypeParameter) _type() {}

func (t *TTypeParameter) GetLocation() ast.Location {
	return t.Location
}

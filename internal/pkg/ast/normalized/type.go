package normalized

import (
	"oak-compiler/internal/pkg/ast"
)

type Type interface {
	_type()
}

type TFunc struct {
	ast.Location
	Params []Type
	Return Type
}

func (*TFunc) _type() {}

type TRecord struct {
	ast.Location
	Fields map[ast.Identifier]Type
}

func (*TRecord) _type() {}

type TTuple struct {
	ast.Location
	Items []Type
}

func (*TTuple) _type() {}

type TUnit struct {
	ast.Location
}

func (*TUnit) _type() {}

type DataOption struct {
	Name   ast.Identifier
	Hidden bool
	Values []Type
}

type TData struct {
	ast.Location
	Name    ast.ExternalIdentifier
	Args    []Type
	Options []DataOption
}

func (*TData) _type() {}

type TExternal struct {
	ast.Location
	Name ast.ExternalIdentifier
	Args []Type
}

func (*TExternal) _type() {}

type TTypeParameter struct {
	ast.Location
	Name ast.Identifier
}

func (*TTypeParameter) _type() {}

func (p *TTypeParameter) String() string {
	return string(p.Name)
}

type TPlaceholder struct {
	Name ast.ExternalIdentifier
}

func (p *TPlaceholder) _type() {}

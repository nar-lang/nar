package parsed

import "oak-compiler/ast"

type Type interface {
	_type()
}

type TFunc struct {
	ast.Location
	Params []Type
	Return Type
}

func (TFunc) _type() {}

type TRecord struct {
	ast.Location
	Fields map[ast.Identifier]Type
}

func (TRecord) _type() {}

type TTuple struct {
	ast.Location
	Items []Type
}

func (TTuple) _type() {}

type TUnit struct {
	ast.Location
}

func (TUnit) _type() {}

type TNamed struct {
	ast.Location
	Name ast.QualifiedIdentifier
	Args []Type
}

func (TNamed) _type() {}

type TData struct {
	ast.Location
	Name   ast.ExternalIdentifier
	Args   []Type
	Values []ast.Identifier
}

func (TData) _type() {}

type TExternal struct {
	ast.Location
	Name ast.ExternalIdentifier
	Args []Type
}

func (TExternal) _type() {}

type TTypeParameter struct {
	ast.Location
	Name ast.Identifier
}

func (TTypeParameter) _type() {}

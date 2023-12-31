package parsed

import (
	"nar-compiler/internal/pkg/ast"
)

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

func (TData) _type() {}

type TNative struct {
	ast.Location
	Name ast.FullIdentifier
	Args []Type
}

func (TNative) _type() {}

type TTypeParameter struct {
	ast.Location
	Name ast.Identifier
}

func (TTypeParameter) _type() {}

package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

type Type interface {
	Statement
	normalize(
		modules map[ast.QualifiedIdentifier]*Module, module *Module, typeModule *Module, namedTypes namedTypeMap,
	) (normalized.Type, error)
	Successor() normalized.Type
	setSuccessor(p normalized.Type)
}

type typeBase struct {
	location  ast.Location
	successor normalized.Type
}

func newTypeBase(loc ast.Location) *typeBase {
	return &typeBase{
		location: loc,
	}
}

func (t *typeBase) GetLocation() ast.Location {
	return t.location
}

func (*typeBase) _parsed() {}

func (t *typeBase) Successor() normalized.Type {
	return t.successor
}

func (t *typeBase) setSuccessor(p normalized.Type) {
	t.successor = p
}

type TFunc struct {
	*typeBase
	params  []Type
	return_ Type
}

func NewTFunc(loc ast.Location, params []Type, ret Type) Type {
	return &TFunc{
		typeBase: newTypeBase(loc),
		params:   params,
		return_:  ret,
	}
}

type TRecord struct {
	*typeBase
	fields map[ast.Identifier]Type
}

func (t *TRecord) Fields() map[ast.Identifier]Type {
	return t.fields
}

func NewTRecord(loc ast.Location, fields map[ast.Identifier]Type) Type {
	return &TRecord{
		typeBase: newTypeBase(loc),
		fields:   fields,
	}
}

type TTuple struct {
	*typeBase
	items []Type
}

func NewTTuple(loc ast.Location, items []Type) Type {
	return &TTuple{
		typeBase: newTypeBase(loc),
		items:    items,
	}
}

type TUnit struct {
	*typeBase
}

func NewTUnit(loc ast.Location) Type {
	return &TUnit{
		typeBase: newTypeBase(loc),
	}
}

type TNamed struct {
	*typeBase
	name ast.QualifiedIdentifier
	args []Type
}

func NewTNamed(loc ast.Location, name ast.QualifiedIdentifier, args []Type) Type {
	return &TNamed{
		typeBase: newTypeBase(loc),
		name:     name,
		args:     args,
	}
}

func (t *TNamed) Find(
	modules map[ast.QualifiedIdentifier]*Module, module *Module,
) (Type, *Module, []ast.FullIdentifier, error) {
	return findType(modules, module, t.name, t.args, t.location)
}

type TData struct {
	*typeBase
	name    ast.FullIdentifier
	args    []Type
	options []DataOption
}

func NewTData(loc ast.Location, name ast.FullIdentifier, args []Type, options []DataOption) Type {
	return &TData{
		typeBase: newTypeBase(loc),
		name:     name,
		args:     args,
		options:  options,
	}
}

type TNative struct {
	*typeBase
	name ast.FullIdentifier
	args []Type
}

func NewTNative(loc ast.Location, name ast.FullIdentifier, args []Type) Type {
	return &TNative{
		typeBase: newTypeBase(loc),
		name:     name,
		args:     args,
	}
}

type TParameter struct {
	*typeBase
	name ast.Identifier
}

func NewTParameter(loc ast.Location, name ast.Identifier) Type {
	return &TParameter{
		typeBase: newTypeBase(loc),
		name:     name,
	}
}

package parsed

import (
	"fmt"
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

type Definition interface {
	Name() string
	getGenerics() GenericParams
	unpackNestedDefinitions() []Definition
	nestedDefinitionNames() []string
	resolve(md *Metadata) (resolvedDefinition resolved.Definition, keepIt bool, err error)
	resolveName(cursor misc.Cursor, md *Metadata) (string, error)
	isHidden() bool
	getAddress() DefinitionAddress
	getType(cursor misc.Cursor, generics GenericArgs, md *Metadata) (Type, GenericArgs, error)
	isExtern() bool
	precondition(md *Metadata) (Definition, error)
}

type Type interface {
	resolve(cursor misc.Cursor, md *Metadata) (resolved.Type, error)
	resolveWithRefName(cursor misc.Cursor, refName string, generics GenericArgs, md *Metadata) (resolved.Type, error)
	dereference(md *Metadata) (Type, error)
	nestedDefinitionNames() []string
	unpackNestedDefinitions(def Definition) []Definition
	mapGenerics(gm genericsMap) Type
	getGenerics() GenericArgs
	equalsTo(other Type, ignoreGenerics bool, md *Metadata) bool
	extractGenerics(other Type) genericsMap
	getCursor() misc.Cursor
	getEnclosingModuleName() ModuleFullName
	extractLocals(type_ Type, md *Metadata) error

	fmt.Stringer
}

type Expression interface {
	precondition(md *Metadata) (Expression, error)
	setType(type_ Type, md *Metadata) (Expression, Type, error)
	getType(md *Metadata) (Type, error)
	resolve(md *Metadata) (resolved.Expression, error)
	getCursor() misc.Cursor
}

type Decons interface {
	resolve(type_ Type, md *Metadata) (resolved.Decons, error)
	extractLocals(type_ Type, md *Metadata) error
	SetAlias(alias string) (Decons, error)
}

type Parameter interface {
	resolve(type_ Type, md *Metadata) (resolved.Parameter, error)
	extractLocals(type_ Type, md *Metadata) error
	SetAlias(alias string) (Parameter, error)
}

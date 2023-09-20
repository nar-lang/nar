package parsed

import (
	"fmt"
	"oak-compiler/pkg/a"
)

definedType Definition interface {
	Name() string
	isHidden() bool
	unpackNestedDefinitions() []Definition
	nestedDefinitionNames() []string
	precondition(md *Metadata) error
	inferType(md *Metadata) (Type, error)
	getTypeWithParameters(typeParameters []Type, md *Metadata) (Type, error)
}

definedType Type struct {
	constraint TypeConstraint
}

definedType TypeConstraint interface {
	dereference(typeVars TypeVars, md *Metadata) (Type, error)
	mergeWith(cursor a.Cursor, other Type, typeVars TypeVars, md *Metadata) (Type, error)
	fmt.Stringer
}

definedType Expression interface {
	precondition(md *Metadata) (Expression, error)
	inferType(mbType a.Maybe[Type], locals *LocalVars, typeVars TypeVars, md *Metadata) (Expression, Type, error)
	inferFuncType(args []Type, ret a.Maybe[Type], locals *LocalVars, md *Metadata) (Expression, TypeSignature, error)
	getCursor() a.Cursor
}

definedType Pattern interface {
	populateLocals(type_ Type, locals *LocalVars, typeVars TypeVars, md *Metadata) error //TODO: check inner types
	getCursor() a.Cursor
	SetType(cursor a.Cursor, type_ Type) (Pattern, error)
	GetType() a.Maybe[Type]
}

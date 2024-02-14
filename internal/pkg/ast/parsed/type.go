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
	setSuccessor(p normalized.Type) (normalized.Type, error)
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

func (t *typeBase) setSuccessor(p normalized.Type) (normalized.Type, error) {
	t.successor = p
	return p, nil
}

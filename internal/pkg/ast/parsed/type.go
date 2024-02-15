package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

type Type interface {
	Statement
	normalize(modules map[ast.QualifiedIdentifier]*Module, module *Module, namedTypes namedTypeMap) (normalized.Type, error)
	setSuccessor(p normalized.Type) (normalized.Type, error)
	applyArgs(params map[ast.Identifier]Type, loc ast.Location) (Type, error)
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

func (t *typeBase) Location() ast.Location {
	return t.location
}

func (*typeBase) _parsed() {}

func (t *typeBase) Successor() normalized.Statement {
	return t.successor
}

func (t *typeBase) setSuccessor(p normalized.Type) (normalized.Type, error) {
	t.successor = p
	return p, nil
}

package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

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

func (t *TParameter) normalize(
	modules map[ast.QualifiedIdentifier]*Module, module *Module, typeModule *Module, namedTypes namedTypeMap,
) (normalized.Type, error) {
	return t.setSuccessor(normalized.NewTParameter(t.location, t.name))
}

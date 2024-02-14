package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

type TUnit struct {
	*typeBase
}

func NewTUnit(loc ast.Location) Type {
	return &TUnit{
		typeBase: newTypeBase(loc),
	}
}

func (t *TUnit) normalize(
	modules map[ast.QualifiedIdentifier]*Module, module *Module, typeModule *Module, namedTypes namedTypeMap,
) (normalized.Type, error) {
	return t.setSuccessor(normalized.NewTUnit(t.location))
}

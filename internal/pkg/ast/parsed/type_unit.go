package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

func NewTUnit(loc ast.Location) Type {
	return &TUnit{
		typeBase: newTypeBase(loc),
	}
}

type TUnit struct {
	*typeBase
}

func (t *TUnit) Iterate(f func(statement Statement)) {
	f(t)
}

func (t *TUnit) normalize(modules map[ast.QualifiedIdentifier]*Module, module *Module, namedTypes namedTypeMap) (normalized.Type, error) {
	return t.setSuccessor(normalized.NewTUnit(t.location))
}

func (t *TUnit) applyArgs(params map[ast.Identifier]Type, loc ast.Location) (Type, error) {
	return t, nil
}

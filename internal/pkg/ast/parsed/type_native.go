package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

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

func (t *TNative) normalize(
	modules map[ast.QualifiedIdentifier]*Module, module *Module, typeModule *Module, namedTypes namedTypeMap,
) (normalized.Type, error) {
	var args []normalized.Type
	for _, arg := range t.args {
		nArg, err := arg.normalize(modules, module, typeModule, namedTypes)
		if err != nil {
			return nil, err
		}
		args = append(args, nArg)
	}
	return t.setSuccessor(normalized.NewTNative(t.location, t.name, args))
}

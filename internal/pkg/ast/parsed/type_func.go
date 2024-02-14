package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
)

type TFunc struct {
	*typeBase
	params  []Type
	return_ Type
}

func NewTFunc(loc ast.Location, params []Type, ret Type) Type {
	if ret == nil && !common.Any(func(x Type) bool { return x != nil }, params) {
		return nil
	}
	return &TFunc{
		typeBase: newTypeBase(loc),
		params:   params,
		return_:  ret,
	}
}

func (t *TFunc) normalize(
	modules map[ast.QualifiedIdentifier]*Module, module *Module, typeModule *Module, namedTypes namedTypeMap,
) (normalized.Type, error) {
	var params []normalized.Type
	for _, param := range t.params {
		if param == nil {
			return nil, common.NewError(t.location, "missing parameter type annotation")
		}
		nParam, err := param.normalize(modules, module, typeModule, namedTypes)
		if err != nil {
			return nil, err
		}
		params = append(params, nParam)
	}
	ret, err := t.return_.normalize(modules, module, typeModule, namedTypes)
	if err != nil {
		return nil, err
	}
	return t.setSuccessor(normalized.NewTFunc(t.location, params, ret))
}

package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
)

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

type TFunc struct {
	*typeBase
	params  []Type
	return_ Type
}

func (t *TFunc) Iterate(f func(statement Statement)) {
	f(t)
	for _, param := range t.params {
		if param != nil {
			param.Iterate(f)
		}
	}
	if t.return_ != nil {
		t.return_.Iterate(f)
	}
}

func (t *TFunc) normalize(modules map[ast.QualifiedIdentifier]*Module, module *Module, namedTypes namedTypeMap) (normalized.Type, error) {
	var params []normalized.Type
	for _, param := range t.params {
		if param == nil {
			return nil, common.NewError(t.location, "missing parameter type annotation")
		}
		nParam, err := param.normalize(modules, module, namedTypes)
		if err != nil {
			return nil, err
		}
		params = append(params, nParam)
	}
	ret, err := t.return_.normalize(modules, module, namedTypes)
	if err != nil {
		return nil, err
	}
	return t.setSuccessor(normalized.NewTFunc(t.location, params, ret))
}

func (t *TFunc) applyArgs(params map[ast.Identifier]Type, loc ast.Location) (Type, error) {
	var fnParams []Type
	for _, param := range t.params {
		fnParam, err := param.applyArgs(params, loc)
		if err != nil {
			return nil, err
		}
		fnParams = append(fnParams, fnParam)
	}

	return_, err := t.return_.applyArgs(params, loc)
	if err != nil {
		return nil, err
	}
	return NewTFunc(loc, fnParams, return_), nil
}

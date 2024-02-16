package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
)

type TFunc struct {
	*typeBase
	params  []Type
	return_ Type
}

func NewTFunc(loc ast.Location, params []Type, ret Type) Type {
	return &TFunc{
		typeBase: newTypeBase(loc),
		params:   params,
		return_:  ret,
	}
}

func (e *TFunc) annotate(ctx *typed.SolvingContext, params typeParamsMap, source bool, placeholders placeholderMap) (typed.Type, error) {
	funcParams, err := common.MapError(func(t Type) (typed.Type, error) {
		if t == nil {
			return nil, common.NewErrorOf(e, "function parameter type is not declared")
		}
		return t.annotate(ctx, params, source, placeholders)
	}, e.params)
	if err != nil {
		return nil, err
	}
	if e.return_ == nil {
		return nil, common.NewErrorOf(e, "function return type is not declared")
	}
	return_, err := e.return_.annotate(ctx, params, source, placeholders)
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(typed.NewTFunc(e.location, funcParams, return_))
}

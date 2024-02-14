package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
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

func (e *TNative) annotate(ctx *typed.SolvingContext, params typeParamsMap, source bool, placeholders placeholderMap) (typed.Type, error) {
	args, err := common.MapError(func(t Type) (typed.Type, error) {
		if t == nil {
			return nil, common.Error{Location: e.location, Message: "type parameter is not declared"}
		}
		return t.annotate(ctx, params, source, placeholders)
	}, e.args)
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(typed.NewTNative(e.location, e.name, args))
}

package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
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

func (e *TParameter) annotate(ctx *typed.SolvingContext, params typeParamsMap, source bool, placeholders placeholderMap) (typed.Type, error) {
	if id, ok := params[e.name]; ok {
		return e.setSuccessor(id)
	} else {
		if source {
			r := typed.NewTParameter(ctx, e.location, e, e.name)
			params[e.name] = r
			return e.setSuccessor(r)
		} else {
			return nil, common.Error{
				Location: e.location, Message: "unknown type parameter",
			}
		}
	}
}

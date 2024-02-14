package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
)

type TRecord struct {
	*typeBase
	fields map[ast.Identifier]Type
}

func NewTRecord(loc ast.Location, fields map[ast.Identifier]Type) Type {
	return &TRecord{
		typeBase: newTypeBase(loc),
		fields:   fields,
	}
}

func (e *TRecord) annotate(ctx *typed.SolvingContext, params typeParamsMap, source bool, placeholders placeholderMap) (typed.Type, error) {
	fields := map[ast.Identifier]typed.Type{}
	for n, v := range e.fields {
		if v == nil {
			return nil, common.Error{Location: e.location, Message: "record field type is not declared"}
		}
		var err error
		fields[n], err = v.annotate(ctx, params, source, placeholders)
		if err != nil {
			return nil, err
		}
	}
	return e.setSuccessor(typed.NewTRecord(e.location, fields, false))
}

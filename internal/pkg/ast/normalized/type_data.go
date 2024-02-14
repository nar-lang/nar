package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
)

type TData struct {
	*typeBase
	name    ast.FullIdentifier
	args    []Type
	options []*DataOption
}

func NewTData(loc ast.Location, name ast.FullIdentifier, args []Type, options []*DataOption) Type {
	return &TData{
		typeBase: newTypeBase(loc),
		name:     name,
		args:     args,
		options:  options,
	}
}

func (e *TData) annotate(ctx *typed.SolvingContext, params typeParamsMap, source bool, placeholders placeholderMap) (typed.Type, error) {
	if placeholders == nil {
		placeholders = placeholderMap{}
	}
	args, err := common.MapError(func(t Type) (typed.Type, error) {
		if t == nil {
			return nil, common.Error{Location: e.location, Message: "type parameter is not declared"}
		}
		return t.annotate(ctx, params, source, placeholders)
	}, e.args)
	if err != nil {
		return nil, err
	}
	annotatedData := typed.NewTData(e.location, e.name, args, nil).(*typed.TData)
	placeholders[e.name] = annotatedData
	options, err := common.MapError(
		func(x *DataOption) (*typed.DataOption, error) {
			values, err := common.MapError(func(t Type) (typed.Type, error) {
				if t == nil {
					return nil, common.Error{Location: e.location, Message: "option value type is not declared"}
				}
				return t.annotate(ctx, params, source, placeholders)
			}, x.values)
			if err != nil {
				return nil, err
			}
			return typed.NewDataOption(common.MakeDataOptionIdentifier(e.name, x.name), values), nil
		},
		e.options)
	if err != nil {
		return nil, err
	}
	annotatedData.SetOptions(options)
	return e.setSuccessor(annotatedData)
}

type DataOption struct {
	name   ast.Identifier
	hidden bool
	values []Type
}

func NewDataOption(name ast.Identifier, hidden bool, values []Type) *DataOption {
	return &DataOption{
		name:   name,
		hidden: hidden,
		values: values,
	}
}

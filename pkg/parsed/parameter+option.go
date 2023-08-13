package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
	"strconv"
)

func NewOptionParameter(c misc.Cursor, index int, optionName string, value Parameter) Parameter {
	return parameterOption{cursor: c, optionName: optionName, value: value, index: index}
}

type parameterOption struct {
	ParameterOption__ int
	cursor            misc.Cursor
	optionName        string
	value             Parameter
	index             int
	alias             string
}

func (p parameterOption) resolve(type_ Type, md *Metadata) (resolved.Parameter, error) {
	dt, err := type_.dereference(md)
	if err != nil {
		return nil, err
	}

	union, ok := dt.(typeUnion)
	if !ok {
		return nil, misc.NewError(p.cursor, "expected union type")
	}

	if len(union.Options) != 0 {
		return nil, misc.NewError(
			p.cursor, "parameter is not exhaustive, only unions with one option can be deconstructed in parameter",
		)
	}
	if p.alias == "" {
		p.alias = "_p" + strconv.Itoa(p.index)
	}
	resolvedType, err := type_.resolve(p.cursor, md)
	if err != nil {
		return nil, err
	}

	valueParam, err := p.value.resolve(union.Options[0].valueType, md)

	return resolved.NewOptionParameter(p.alias, resolvedType, valueParam), nil
}

func (p parameterOption) extractLocals(type_ Type, md *Metadata) error {
	panic("not implemented")
	return nil
}

func (p parameterOption) SetAlias(alias string) (Parameter, error) {
	return nil, misc.NewError(p.cursor, "named parameter cannot have alias")
}

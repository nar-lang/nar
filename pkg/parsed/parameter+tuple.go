package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
	"strconv"
)

func NewTupleParameter(c misc.Cursor, index int, items []Parameter) Parameter {
	return parameterTuple{cursor: c, index: index, items: items}
}

type parameterTuple struct {
	ParameterNamed__ int
	cursor           misc.Cursor
	alias            string
	index            int
	items            []Parameter
}

func (p parameterTuple) resolve(type_ Type, md *Metadata) (resolved.Parameter, error) {
	if p.alias == "" {
		p.alias = "_p" + strconv.Itoa(p.index)
	}
	dt, err := type_.dereference(md)
	if err != nil {
		return nil, err
	}
	tuple, ok := dt.(typeTuple)
	if !ok {
		return nil, misc.NewError(p.cursor, "expected tuple got %s", type_)
	}

	if len(tuple.Items) != len(p.items) {
		return nil, misc.NewError(p.cursor, "expected %d-tuple got %d-tuple", len(tuple.Items), len(p.items))
	}

	resolvedType, err := type_.resolve(p.cursor, md)
	if err != nil {
		return nil, err
	}
	var resolvedItems []resolved.Parameter
	for i, itemType := range tuple.Items {
		ri, err := p.items[i].resolve(itemType, md)
		if err != nil {
			return nil, err
		}
		resolvedItems = append(resolvedItems, ri)
	}

	return resolved.NewTupleParameter(p.alias, resolvedType, resolvedItems), nil
}

func (p parameterTuple) extractLocals(type_ Type, md *Metadata) error {

	dt, err := type_.dereference(md)
	if err != nil {
		return err
	}
	tuple, ok := dt.(typeTuple)
	if !ok {
		return misc.NewError(p.cursor, "expected tuple got %s", type_)
	}
	if len(tuple.Items) != len(p.items) {
		return misc.NewError(p.cursor, "expected %d-tuple got %d-tuple", len(tuple.Items), len(p.items))
	}
	for i, item := range p.items {
		err = item.extractLocals(tuple.Items[i], md)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p parameterTuple) SetAlias(alias string) (Parameter, error) {
	p.alias = alias
	return p, nil
}

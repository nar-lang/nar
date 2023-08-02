package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewTupleDecons(c misc.Cursor, items []Decons, alias string) Decons {
	return deconsTuple{cursor: c, Items: items, Alias: alias}
}

type deconsTuple struct {
	DeconsTuple__ int
	Items         []Decons
	Alias         string
	cursor        misc.Cursor
}

func (d deconsTuple) extractLocals(type_ Type, md *Metadata) error {
	if d.Alias != "" {
		md.LocalVars[d.Alias] = type_
	}

	dt, err := type_.dereference(md)
	if err != nil {
		return err
	}
	tuple, ok := dt.(typeTuple)
	if !ok {
		return misc.NewError(d.cursor, "expected tuple type")
	}
	if len(tuple.Items) != len(d.Items) {
		return misc.NewError(d.cursor, "expected %d-tuple got %d-tuple", len(tuple.Items), len(d.Items))
	}

	for i, item := range d.Items {
		if err := item.extractLocals(tuple.Items[i], md); err != nil {
			return err
		}
	}
	return nil
}

func (d deconsTuple) resolve(type_ Type, md *Metadata) (resolved.Decons, error) {
	if d.Alias != "" {
		md.LocalVars[d.Alias] = type_
	}

	dt, err := type_.dereference(md)
	if err != nil {
		return nil, err
	}
	tuple, ok := dt.(typeTuple)
	if !ok {
		return nil, misc.NewError(d.cursor, "expected tuple type")
	}
	if len(tuple.Items) != len(d.Items) {
		return nil, misc.NewError(d.cursor, "expected %d-tuple got %d-tuple", len(tuple.Items), len(d.Items))
	}

	var items []resolved.Decons
	for i, item := range d.Items {
		rd, err := item.resolve(tuple.Items[i], md)
		if err != nil {
			return nil, err
		}
		items = append(items, rd)
	}

	return resolved.NewTupleDecons(items, d.Alias), nil
}

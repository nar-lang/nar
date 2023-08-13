package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewOptionDecons(c misc.Cursor, option string, args GenericArgs, arg Decons) Decons {
	return deconsOption{cursor: c, Option: option, GenericArgs: args, Arg: arg}
}

type deconsOption struct {
	DeconsOption__ int
	Option         string
	GenericArgs    GenericArgs
	Arg            Decons
	Alias          string
	cursor         misc.Cursor
}

func (d deconsOption) SetAlias(alias string) (Decons, error) {
	d.Alias = alias
	return d, nil
}

func (d deconsOption) extractLocals(type_ Type, md *Metadata) error {
	if d.Alias != "" {
		md.LocalVars[d.Alias] = type_
	}

	dt, err := type_.dereference(md)
	if err != nil {
		return err
	}
	union, ok := dt.(typeUnion)
	if !ok {
		return misc.NewError(d.cursor, "expected union type")
	}

	for _, o := range union.Options {
		if o.name == d.Option {
			if err := d.Arg.extractLocals(o.valueType, md); err != nil {
				return err
			}
		}
	}

	if !ok {
		return misc.NewError(d.cursor, "union type does not contain this option")
	}
	return nil
}

func (d deconsOption) resolve(type_ Type, md *Metadata) (resolved.Decons, error) {
	if d.Alias != "" {
		md.LocalVars[d.Alias] = type_
	}
	dt, err := type_.dereference(md)
	if err != nil {
		return nil, err
	}
	union, ok := dt.(typeUnion)
	if !ok {
		return nil, misc.NewError(d.cursor, "expected union type")
	}

	for _, o := range union.Options {
		if o.name == d.Option {
			resolvedType, err := o.valueType.resolve(d.cursor, md)
			if err != nil {
				return nil, err
			}
			resolvedArg, err := d.Arg.resolve(o.valueType, md)
			if err != nil {
				return nil, err
			}
			return resolved.NewOptionDecons(d.Option, resolvedType, resolvedArg, d.Alias), nil
		}
	}

	if !ok {
		return nil, misc.NewError(d.cursor, "union type does not contain this option")
	}
	return nil, nil
}

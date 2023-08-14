package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

type expressionOption struct {
	Type     Type
	Address  DefinitionAddress
	Generics GenericArgs
	Option   string
	Value    Expression
	cursor   misc.Cursor
}

func (e expressionOption) getCursor() misc.Cursor {
	return e.cursor
}

func (e expressionOption) precondition(md *Metadata) (Expression, error) {
	var err error
	e.Value, err = e.Value.precondition(md)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (e expressionOption) setType(type_ Type, md *Metadata) (Expression, Type, error) {
	dt, err := type_.dereference(md)
	if err != nil {
		return nil, nil, err
	}
	unionType, ok := dt.(typeUnion)
	if !ok {
		return nil, nil, misc.NewError(e.cursor, "expected union type, got %s", type_)
	}

	gm := e.Type.extractGenerics(unionType)

	for i, o := range unionType.Options {
		if o.name == e.Option {
			var err error
			e.Value, o.valueType, err = e.Value.setType(o.valueType, md)
			if err != nil {
				return nil, nil, err
			}
			unionType.Options[i] = o

			e.Type.mapGenerics(gm)
			return e, e.Type.mapGenerics(gm), nil
		}
	}

	return nil, nil, misc.NewError(e.cursor, "option %s is not declared for union type %s", e.Option, type_)
}

func (e expressionOption) getType(md *Metadata) (Type, error) {
	return e.Type, nil
}

func (e expressionOption) resolve(md *Metadata) (resolved.Expression, error) {
	var resolvedValue resolved.Expression
	var err error
	resolvedValue, err = e.Value.resolve(md)
	if err != nil {
		return nil, err
	}
	t, _ := e.getType(md)
	refName, err := md.makeRefNameByAddress(e.Address, e.cursor)
	if err != nil {
		return nil, err
	}
	resolvedType, err := t.resolve(e.cursor, md)
	if err != nil {
		return nil, err
	}
	resolvedGenerics, err := e.Generics.resolve(e.cursor, md)
	if err != nil {
		return nil, err
	}

	return resolved.NewOptionExpression(resolvedType, resolvedGenerics, refName, e.Option, resolvedValue), nil
}

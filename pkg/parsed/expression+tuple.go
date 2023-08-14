package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewTupleExpression(c misc.Cursor, items []Expression) Expression {
	return expressionTuple{cursor: c, Items: items}
}

type expressionTuple struct {
	Items  []Expression
	cursor misc.Cursor
}

func (e expressionTuple) getCursor() misc.Cursor {
	return e.cursor
}

func (e expressionTuple) precondition(md *Metadata) (Expression, error) {
	var err error
	for i, item := range e.Items {
		e.Items[i], err = item.precondition(md)
		if err != nil {
			return nil, err
		}
	}
	return e, nil
}

func (e expressionTuple) setType(type_ Type, md *Metadata) (Expression, Type, error) {
	tuple, ok := type_.(typeTuple)
	if !ok {
		return nil, nil, misc.NewError(e.cursor, "expected tuple type here")
	}
	var err error
	var inferredItems []Type
	for i, itemType := range tuple.Items {
		var inferredItemType Type
		e.Items[i], inferredItemType, err = e.Items[i].setType(itemType, md)
		inferredItems = append(inferredItems, inferredItemType)
		if err != nil {
			return nil, nil, err
		}
	}
	tuple.Items = inferredItems
	return e, tuple, nil
}

func (e expressionTuple) getType(md *Metadata) (Type, error) {
	var types []Type
	for _, ex := range e.Items {
		t, err := ex.getType(md)
		if err != nil {
			return nil, err
		}
		types = append(types, t)
	}
	return typeTuple{Items: types}, nil
}

func (e expressionTuple) resolve(md *Metadata) (resolved.Expression, error) {
	var expressions []resolved.Expression
	for _, ex := range e.Items {
		resolvedExpression, err := ex.resolve(md)
		if err != nil {
			return nil, err
		}
		expressions = append(expressions, resolvedExpression)
	}

	var types []resolved.Type
	for _, e := range expressions {
		types = append(types, e.Type())
	}

	return resolved.NewTupleExpression(resolved.NewTupleType(types), expressions), nil
}

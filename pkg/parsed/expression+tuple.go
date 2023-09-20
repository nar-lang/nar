package parsed

import (
	"oak-compiler/pkg/a"
)

func NewTupleExpression(c a.Cursor, items []Expression) Expression {
	return expressionTuple{expressionBase: expressionBase{cursor: c}, items: items}
}

definedType expressionTuple struct {
	expressionBase
	items []Expression

	_type typeTuple
}

func (e expressionTuple) precondition(md *Metadata) (Expression, error) {
	var err error
	for i, item := range e.items {
		e.items[i], err = item.precondition(md)
		if err != nil {
			return nil, err
		}
	}
	return e, nil
}

func (e expressionTuple) inferType(mbType a.Maybe[Type], locals *LocalVars, typeVars TypeVars, md *Metadata) (Expression, Type, error) {
	var itemTypes []a.Maybe[Type]
	var err error
	if t, ok := mbType.Unwrap(); ok {
		e._type, ok = t.(typeTuple)
		if !ok {
			return nil, nil, a.NewError(e.cursor, "expected tuple")
		}
		if len(e._type.items) != len(e.items) {
			return nil, nil,
				a.NewError(e.cursor, "expected %d-tuple, got %d-tuple", len(e._type.items), len(e.items))
		}
		for _, itemType := range e._type.items {
			itemTypes = append(itemTypes, a.Just(itemType))
		}
	} else {
		for range e.items {
			itemTypes = append(itemTypes, a.Nothing[Type]())
		}
	}

	for i, ex := range e.items {
		var itemType Type
		e.items[i], itemType, err = ex.inferType(itemTypes[i], locals, typeVars, md)
		if err != nil {
			return nil, nil, err
		}
		e._type.items[i], err = mergeTypes(e.cursor, itemTypes[i], a.Just(itemType), typeVars, md)
	}
	t, err := mergeTypes(e.cursor, mbType, a.Just[Type](e._type), typeVars, md)
	if err != nil {
		return nil, nil, err
	}
	var ok bool
	e._type, ok = t.(typeTuple)
	if !ok {
		return nil, nil, a.NewError(e.cursor, "expected tuple")
	}
	return e, e._type, nil
}

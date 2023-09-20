package parsed

import (
	"oak-compiler/pkg/a"
)

func NewListExpression(c a.Cursor, items []Expression) Expression {
	return expressionList{expressionBase: expressionBase{cursor: c}, items: items}
}

definedType expressionList struct {
	expressionBase
	items []Expression

	_type     Type
	_itemType Type
}

func (e expressionList) precondition(md *Metadata) (Expression, error) {
	var err error
	for i, item := range e.items {
		e.items[i], err = item.precondition(md)
		if err != nil {
			return nil, err
		}
	}
	return e, nil
}

func (e expressionList) inferType(mbType a.Maybe[Type], locals *LocalVars, typeVars TypeVars, md *Metadata) (Expression, Type, error) {
	if t, ok := mbType.Unwrap(); ok {
		var err error
		e._type, e._itemType, err = ExtractListTypeAndItemType(e.cursor, t, typeVars, md)
		if err != nil {
			return nil, nil, err
		}
	} else {
		e._itemType = typeVariable{typeBase: typeBase{cursor: e.cursor}, name: "?0"}
		e._type = TypeBuiltinList(e.cursor, e._itemType)
	}

	for i, ex := range e.items {
		var itemType Type
		var err error
		e.items[i], itemType, err = ex.inferType(a.Just(e._itemType), locals, typeVars, md)
		if err != nil {
			return nil, nil, err
		}
		itemType, err = mergeTypes(ex.getCursor(), a.Just(e._itemType), a.Just(itemType), typeVars, md)
		if err != nil {
			return nil, nil, err
		}
	}
	return e, e._type, nil
}

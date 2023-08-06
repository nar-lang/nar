package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewListExpression(c misc.Cursor, items []Expression) Expression {
	return expressionList{cursor: c, Items: items}
}

type expressionList struct {
	ExpressionList__ int
	Items            []Expression
	ItemType         Type
	cursor           misc.Cursor
}

func (e expressionList) getCursor() misc.Cursor {
	return e.cursor
}

func (e expressionList) precondition(md *Metadata) (Expression, error) {
	var err error
	for i, item := range e.Items {
		e.Items[i], err = item.precondition(md)
		if err != nil {
			return nil, err
		}
	}
	return e, nil
}

func (e expressionList) setType(type_ Type, gm genericsMap, md *Metadata) (Expression, Type, error) {
	gs := type_.getGenerics()
	if len(gs) != 1 {
		return nil, nil, misc.NewError(e.cursor, "expected list type here")
	}
	e.ItemType = gs[0]
	inferredType := TypeBuiltinList(e.cursor, md.currentModuleName(), e.ItemType)
	if !typesEqual(type_, inferredType, false, md) {
		return nil, nil, misc.NewError(e.cursor, "types do not match, expected %s got % s", inferredType, type_)
	}

	var err error
	for i, item := range e.Items {
		e.Items[i], e.ItemType, err = item.setType(e.ItemType, gm, md)
		if err != nil {
			return nil, nil, err
		}
	}
	for i, item := range e.Items {
		e.Items[i], _, err = item.setType(e.ItemType, gm, md)
		if err != nil {
			return nil, nil, err
		}
	}

	return e, TypeBuiltinList(e.cursor, md.currentModuleName(), e.ItemType), nil
}

func (e expressionList) getType(md *Metadata) (Type, error) {
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

func (e expressionList) resolve(md *Metadata) (resolved.Expression, error) {
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

	resolvedList, err := TypeBuiltinList(e.cursor, md.currentModuleName(), e.ItemType).resolve(e.cursor, md)
	if err != nil {
		return nil, err
	}
	resolvedItemType, err := e.ItemType.resolve(e.cursor, md)
	if err != nil {
		return nil, err
	}
	return resolved.NewListExpression(resolvedList, resolvedItemType, expressions), nil
}

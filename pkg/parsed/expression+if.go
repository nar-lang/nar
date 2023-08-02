package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewIfExpression(c misc.Cursor, condition, positive, negative Expression) Expression {
	return expressionIf{cursor: c, Condition: condition, Positive: positive, Negative: negative}
}

type expressionIf struct {
	ExpressionIf__ int
	Condition      Expression
	Positive       Expression
	Negative       Expression
	cursor         misc.Cursor
}

func (e expressionIf) getCursor() misc.Cursor {
	return e.cursor
}

func (e expressionIf) precondition(md *Metadata) (Expression, error) {
	var err error
	e.Condition, err = e.Condition.precondition(md)
	if err != nil {
		return nil, err
	}
	e.Positive, err = e.Positive.precondition(md)
	if err != nil {
		return nil, err
	}
	e.Negative, err = e.Negative.precondition(md)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (e expressionIf) setType(type_ Type, gm genericsMap, md *Metadata) (Expression, Type, error) {
	var err error
	var positiveType, negativeType Type
	e.Condition, _, err = e.Condition.setType(TypeBuiltinBool(e.cursor, md.currentModuleName()), gm, md)
	if err != nil {
		return nil, nil, err
	}
	e.Positive, positiveType, err = e.Positive.setType(type_, gm, md)
	if err != nil {
		return nil, nil, err
	}
	e.Negative, negativeType, err = e.Negative.setType(type_, gm, md)
	if err != nil {
		return nil, nil, err
	}

	if !typesEqual(positiveType, negativeType, false, md) {
		return nil, nil, misc.NewError(e.cursor,
			"positive and negative branches have different types: %s and %s",
			positiveType,
			negativeType)
	}

	return e, positiveType, nil
}

func (e expressionIf) getType(md *Metadata) (Type, error) {
	return e.Positive.getType(md)
}

func (e expressionIf) resolve(md *Metadata) (resolved.Expression, error) {
	resolvedCondition, err := e.Condition.resolve(md)
	if err != nil {
		return nil, err
	}
	resolvedPositive, err := e.Positive.resolve(md)
	if err != nil {
		return nil, err
	}
	resolvedNegative, err := e.Negative.resolve(md)
	if err != nil {
		return nil, err
	}
	return resolved.NewIfExpression(resolvedCondition, resolvedPositive, resolvedNegative), nil
}

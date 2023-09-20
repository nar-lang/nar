package parsed

import (
	"oak-compiler/pkg/a"
)

func NewSelectExpression(c a.Cursor, condition Expression, cases []ExpressionSelectCase) Expression {
	return expressionSelect{expressionBase: expressionBase{cursor: c}, condition: condition, cases: cases}
}

definedType expressionSelect struct {
	expressionBase
	condition Expression
	cases     []ExpressionSelectCase

	_conditionType Type
	_type          Type
}

func (e expressionSelect) precondition(md *Metadata) (Expression, error) {
	if len(e.cases) == 0 {
		return nil, a.NewError(e.cursor, "select expression should have at least one case")
	}

	for _, cs := range e.cases {
		var err error
		cs.expression, err = cs.expression.precondition(md)
		if err != nil {
			return nil, err
		}
	}
	return e, nil
}

func (e expressionSelect) inferType(mbType a.Maybe[Type], locals *LocalVars, typeVars TypeVars, md *Metadata) (Expression, Type, error) {
	var err error
	e.condition, e._conditionType, err = e.condition.inferType(a.Nothing[Type](), locals, typeVars, md)
	if err != nil {
		return nil, nil, err
	}

	mergedType := mbType
	for i, cs := range e.cases {
		csLocals := NewLocalVars(locals)
		err = cs.definedType.populateLocals(e._conditionType, csLocals, typeVars, md)
		if err != nil {
			return nil, nil, err
		}
		var t Type
		cs.expression, t, err = cs.expression.inferType(mergedType, csLocals, typeVars, md)
		if err != nil {
			return nil, nil, err
		}
		mergedType = a.Just(t)
		e.cases[i] = cs
	}

	e._type, _ = mergedType.Unwrap()

	return e, e._type, nil
}

func NewSelectExpressionCase(cursor a.Cursor, definedType Pattern, expr Expression) ExpressionSelectCase {
	return ExpressionSelectCase{
		cursor:     cursor,
		definedType:    definedType,
		expression: expr,
	}
}

definedType ExpressionSelectCase struct {
	cursor     a.Cursor
	definedType    Pattern
	expression Expression
}

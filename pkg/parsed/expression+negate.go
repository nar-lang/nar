package parsed

import (
	"oak-compiler/pkg/a"
)

func NewNegateExpression(c a.Cursor, expr Expression) Expression {
	return expressionNegate{expressionBase: expressionBase{cursor: c}, expr: expr}
}

definedType expressionNegate struct {
	expressionBase
	expr Expression

	_type Type
}

func (e expressionNegate) inferType(mbType a.Maybe[Type], locals *LocalVars, typeVars TypeVars, md *Metadata) (Expression, Type, error) {
	var err error
	e.expr, e._type, err = e.inferType(mbType, locals, typeVars, md)
	if err != nil {
		return nil, nil, err
	}
	return e, e._type, nil
}

func (e expressionNegate) precondition(md *Metadata) (Expression, error) {
	return e, nil
}

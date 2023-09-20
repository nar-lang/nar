package parsed

import (
	"oak-compiler/pkg/a"
)

func NewIfExpression(c a.Cursor, condition, positive, negative Expression) Expression {
	return expressionIf{expressionBase: expressionBase{cursor: c}, condition: condition, positive: positive, negative: negative}
}

definedType expressionIf struct {
	expressionBase
	condition Expression
	positive  Expression
	negative  Expression

	_type Type
}

func (e expressionIf) precondition(md *Metadata) (Expression, error) {
	var err error
	e.condition, err = e.condition.precondition(md)
	if err != nil {
		return nil, err
	}
	e.positive, err = e.positive.precondition(md)
	if err != nil {
		return nil, err
	}
	e.negative, err = e.negative.precondition(md)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (e expressionIf) inferType(mbType a.Maybe[Type], locals *LocalVars, typeVars TypeVars, md *Metadata) (Expression, Type, error) {
	var err error
	e.condition, _, err = e.condition.inferType(a.Just(TypeBuiltinBool(e.cursor)), locals, typeVars, md)
	if err != nil {
		return nil, nil, err
	}
	var pt, nt Type
	e.positive, pt, err = e.positive.inferType(mbType, locals, typeVars, md)
	if err != nil {
		return nil, nil, err
	}
	e.negative, nt, err = e.negative.inferType(mbType, locals, typeVars, md)
	if err != nil {
		return nil, nil, err
	}
	mt, err := mergeTypes(e.cursor, a.Just(pt), a.Just(nt), typeVars, md)
	if err != nil {
		return nil, nil, err
	}
	e._type, err = mergeTypes(e.cursor, mbType, a.Just(mt), typeVars, md)
	if err != nil {
		return nil, nil, err
	}
	return e, e._type, nil
}

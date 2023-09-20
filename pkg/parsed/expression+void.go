package parsed

import (
	"oak-compiler/pkg/a"
)

func NewVoidExpression(c a.Cursor) Expression {
	return expressionVoid{expressionBase: expressionBase{cursor: c}}
}

definedType expressionVoid struct {
	expressionBase
}

func (e expressionVoid) precondition(md *Metadata) (Expression, error) {
	return e, nil
}

func (e expressionVoid) inferType(mbType a.Maybe[Type], locals *LocalVars, typeVars TypeVars, md *Metadata) (Expression, Type, error) {
	t, err := mergeTypes(e.cursor, mbType, a.Just(NewVoidType(e.cursor)), typeVars, md)
	if err != nil {
		return nil, nil, err
	}
	return e, t, nil
}

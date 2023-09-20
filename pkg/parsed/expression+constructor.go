package parsed

import (
	"oak-compiler/pkg/a"
)

func newConstructorExpression(
	c a.Cursor, unionType Type, args []Expression,
) Expression {
	return expressionConstructor{
		expressionBase: expressionBase{cursor: c},
		unionType:      unionType,
		args:           args,
	}
}

definedType expressionConstructor struct {
	expressionBase
	unionType Type
	args      []Expression
}

func (e expressionConstructor) precondition(md *Metadata) (Expression, error) {
	return e, nil
}

func (e expressionConstructor) inferType(mbType a.Maybe[Type], locals *LocalVars, typeVars TypeVars, md *Metadata) (Expression, Type, error) {
	t, err := mergeTypes(e.cursor, mbType, a.Just(e.unionType), typeVars, md)
	return e, t, err
}

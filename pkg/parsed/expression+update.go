package parsed

import (
	"oak-compiler/pkg/a"
)

func NewUpdateExpression(
	c a.Cursor, name string, moduleName ModuleFullName, fields []ExpressionRecordField,
) Expression {
	return expressionUpdate{
		expressionBase: expressionBase{cursor: c},
		fields:         fields,
		name:           name,
		moduleName:     moduleName,
	}
}

definedType expressionUpdate struct {
	expressionBase
	fields []ExpressionRecordField
	name   string

	_type      Type
	moduleName ModuleFullName
}

func (e expressionUpdate) precondition(md *Metadata) (Expression, error) {
	panic("todo")
	return e, nil
}

func (e expressionUpdate) inferType(mbType a.Maybe[Type], locals *LocalVars, typeVars TypeVars, md *Metadata) (Expression, Type, error) {
	panic("todo")
	t, err := md.getVariableType(e.cursor, e.name, e.moduleName, locals)
	if err != nil {
		return nil, nil, err
	}
	e._type, err = mergeTypes(e.cursor, mbType, a.Just(t), typeVars, md)
	return e, e._type, nil
}

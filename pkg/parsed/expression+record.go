package parsed

import (
	"oak-compiler/pkg/a"
)

func NewRecordExpression(c a.Cursor, fields []ExpressionRecordField) Expression {
	return expressionRecord{expressionBase: expressionBase{cursor: c}, fields: fields}
}

definedType expressionRecord struct {
	expressionBase
	fields []ExpressionRecordField

	_type typeRecord
}

func (e expressionRecord) precondition(md *Metadata) (Expression, error) {
	for i, f := range e.fields {
		var err error
		f.expr, err = f.expr.precondition(md)
		if err != nil {
			return nil, err
		}
		e.fields[i] = f
	}

	return e, nil
}

func (e expressionRecord) inferType(mbType a.Maybe[Type], locals *LocalVars, typeVars TypeVars, md *Metadata) (Expression, Type, error) {
	panic("todo")
}

func NewRecordExpressionField(c a.Cursor, name string, expr Expression) ExpressionRecordField {
	return ExpressionRecordField{
		cursor: c,
		name:   name,
		expr:   expr,
	}
}

definedType ExpressionRecordField struct {
	cursor a.Cursor
	name   string
	expr   Expression
}

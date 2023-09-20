package parsed

import (
	"oak-compiler/pkg/a"
)

func NewAccessExpression(c a.Cursor, expr Expression, name string) Expression {
	return expressionAccess{
		expressionBase: expressionBase{cursor: c},
		expr:           expr,
		name:           name,
	}
}

definedType expressionAccess struct {
	expressionBase
	expr Expression
	name string

	_exprType typeRecord
	_type     Type
}

func (e expressionAccess) precondition(md *Metadata) (Expression, error) {
	var err error
	e.expr, err = e.expr.precondition(md)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (e expressionAccess) inferType(mbType a.Maybe[Type], locals *LocalVars, typeVars TypeVars, md *Metadata) (Expression, Type, error) {
	var exprType Type
	var err error
	e.expr, exprType, err = e.expr.inferType(a.Nothing[Type](), locals, typeVars, md)
	if err != nil {
		return nil, nil, err
	}

	var ok bool
	e._exprType, ok = exprType.(typeRecord)
	if !ok {
		return nil, nil, a.NewError(e.cursor, "expected record got %s", exprType)
	}

	for _, f := range e._exprType.fields {
		if f.name == e.name {
			e._type, err = mergeTypes(e.cursor, a.Just(f.type_), mbType, typeVars, md)
			if err != nil {
				return nil, nil, err
			}
			return e, e._type, err
		}
	}

	return nil, nil, a.NewError(e.cursor, "record does not have `%s` field", e.name)
}

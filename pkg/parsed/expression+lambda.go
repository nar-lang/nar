package parsed

import (
	"fmt"
	"oak-compiler/pkg/a"
)

func NewLambdaExpression(c a.Cursor, params []Pattern, body Expression) Expression {
	return expressionLambda{expressionBase: expressionBase{cursor: c}, params: params, body: body}
}

definedType expressionLambda struct {
	expressionBase
	params []Pattern
	body   Expression

	_signature TypeSignature
}

func (e expressionLambda) precondition(md *Metadata) (Expression, error) {
	return e, nil
}

func (e expressionLambda) inferType(mbType a.Maybe[Type], locals *LocalVars, typeVars TypeVars, md *Metadata) (Expression, Type, error) {
	if t, ok := mbType.Unwrap(); ok {
		if e._signature, ok = t.(TypeSignature); !ok {
			return nil, nil, a.NewError(e.cursor, "expected function got %s", t)
		}
		if len(e._signature.paramTypes) != len(e.params) {
			return nil, nil, a.NewError(
				e.cursor,
				"expected function with %d parameters, got with %d",
				len(e._signature.paramTypes),
				len(e.params),
			)
		}
	} else {
		var paramTypes []Type
		for i := range e.params {
			paramTypes = append(paramTypes, NewVariableType(e.cursor, fmt.Sprintf("?@%d", i)))
		}

		e._signature = NewSignatureType(e.cursor, paramTypes, NewVariableType(e.cursor, "@@"))
	}

	locals = NewLocalVars(locals)
	for i, p := range e.params {
		err := p.populateLocals(e._signature.paramTypes[i], locals, typeVars, md)
		if err != nil {
			return nil, nil, err
		}
	}

	return e, e._signature, nil
}

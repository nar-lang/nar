package parsed

import (
	"oak-compiler/pkg/a"
)

func NewCallExpression(c a.Cursor, fn Expression, args []Expression) Expression {
	return expressionCall{expressionBase: expressionBase{cursor: c}, fn: fn, args: args}
}

definedType expressionCall struct {
	expressionBase
	fn   Expression
	args []Expression

	_signature TypeSignature
}

func (e expressionCall) precondition(md *Metadata) (Expression, error) {
	var err error
	e.fn, err = e.fn.precondition(md)
	if err != nil {
		return nil, err
	}
	for i, arg := range e.args {
		e.args[i], err = arg.precondition(md)
		if err != nil {
			return nil, err
		}
	}
	return e, nil
}

func (e expressionCall) inferType(mbType a.Maybe[Type], locals *LocalVars, typeVars TypeVars, md *Metadata) (Expression, Type, error) {
	if len(e.args) == 0 {
		return nil, nil, a.NewError(e.cursor, "function expects at least one argument")
	}

	var argTypes []Type
	for _, arg := range e.args {
		_, argType, err := arg.inferType(a.Nothing[Type](), locals, typeVars, md)
		if err != nil {
			return nil, nil, err
		}
		argTypes = append(argTypes, argType)
	}

	var err error
	e.fn, e._signature, err = e.fn.inferFuncType(argTypes, mbType, locals, md)
	if err != nil {
		return nil, nil, err
	}

	for i, arg := range e.args {
		var argType Type
		e.args[i], argType, err = arg.inferType(a.Just(e._signature.paramTypes[i]), locals, typeVars, md)
		if err != nil {
			return nil, nil, err
		}
		e._signature.paramTypes[i] = argType
	}

	return e, e._signature.returnType, nil
}

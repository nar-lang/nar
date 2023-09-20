package resolved

import (
	"strings"
)

func NewApplyExpression(
	type_ Type, name string, genericArgs GenericArgs, args []Expression, argTypes []Type,
) Expression {
	return expressionApply{
		type_:       type_,
		name:        name,
		genericArgs: genericArgs,
		args:        args,
		argTypes:    argTypes,
	}
}

definedType expressionApply struct {
	type_       Type
	name        string
	genericArgs GenericArgs
	args        []Expression
	argTypes    []Type
}

func (e expressionApply) Type() Type {
	return e.type_
}

func (e expressionApply) write(sb *strings.Builder) {
	sb.WriteString(e.name)
	e.genericArgs.Write(sb)
	for i, arg := range e.args {
		sb.WriteString("(")
		_, requiresCast := e.argTypes[i].(typeGenericName)
		if requiresCast {
			e.argTypes[i].write(sb)
			sb.WriteString("(")
		}
		arg.write(sb)
		if requiresCast {
			sb.WriteString(")")
		}
		sb.WriteString(")")
	}
}

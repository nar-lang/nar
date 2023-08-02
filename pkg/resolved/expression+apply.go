package resolved

import (
	"strings"
)

func NewApplyExpression(type_ Type, name string, genericArgs GenericArgs, args []Expression) Expression {
	return expressionApply{
		type_:       type_,
		name:        name,
		genericArgs: genericArgs,
		args:        args,
	}
}

type expressionApply struct {
	type_       Type
	name        string
	genericArgs GenericArgs
	args        []Expression
}

func (e expressionApply) Type() Type {
	return e.type_
}

func (e expressionApply) write(sb *strings.Builder) {
	sb.WriteString(e.name)
	e.genericArgs.Write(sb)
	for _, arg := range e.args {
		sb.WriteString("(")
		arg.write(sb)
		sb.WriteString(")")
	}
}

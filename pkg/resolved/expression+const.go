package resolved

import "strings"

func NewConstExpression(type_ Type, value string) Expression {
	return expressionConst{type_: type_, value: value}
}

type expressionConst struct {
	type_ Type
	value string
}

func (e expressionConst) Type() Type {
	return e.type_
}

func (e expressionConst) write(sb *strings.Builder) {
	if e.value != "" {
		e.type_.write(sb)
		sb.WriteString("(")
		sb.WriteString(e.value)
		sb.WriteString(")")
	}
}

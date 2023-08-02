package resolved

import "strings"

func NewOptionExpression(
	type_ Type, args GenericArgs, typeName string, option string, value Expression,
) Expression {
	return expressionOption{
		type_:       type_,
		genericArgs: args,
		typeName:    typeName,
		option:      option,
		value:       value,
	}
}

type expressionOption struct {
	type_       Type
	genericArgs GenericArgs
	typeName    string
	option      string
	value       Expression
}

func (e expressionOption) Type() Type {
	return e.type_
}

func (e expressionOption) write(sb *strings.Builder) {
	sb.WriteString(e.typeName)
	e.genericArgs.Write(sb)
	sb.WriteString("{")
	sb.WriteString("Value: ")
	e.value.write(sb)
	sb.WriteString(", Option: \"")
	sb.WriteString(e.option)
	sb.WriteString("\"}")
}

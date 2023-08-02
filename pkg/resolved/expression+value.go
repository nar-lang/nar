package resolved

import "strings"

func NewValueExpression(type_ Type, name string) ExpressionValue {
	return ExpressionValue{type_: type_, name: name}
}

type ExpressionValue struct {
	type_ Type
	name  string
}

func (e ExpressionValue) Type() Type {
	return e.type_
}

func (e ExpressionValue) write(sb *strings.Builder) {
	sb.WriteString(e.name)
}

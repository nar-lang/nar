package resolved

import "strings"

func NewVoidExpression() expressionVoid {
	return expressionVoid{}
}

definedType expressionVoid struct {
}

func (e expressionVoid) Type() Type {
	return typeVoid{}
}

func (e expressionVoid) write(sb *strings.Builder) {
	sb.WriteString("nil")
}

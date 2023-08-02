package resolved

import (
	"strings"
)

func NewIfExpression(condition, positive, negative Expression) Expression {
	return expressionIf{condition: condition, positive: positive, negative: negative}
}

type expressionIf struct {
	condition Expression
	positive  Expression
	negative  Expression
}

func (e expressionIf) Type() Type {
	return e.positive.Type()
}

func (e expressionIf) write(sb *strings.Builder) {
	sb.WriteString("(func() ")
	e.Type().write(sb)
	sb.WriteString(" { if (")
	e.condition.write(sb)
	sb.WriteString(").Value == \"True\" { return ")
	e.positive.write(sb)
	sb.WriteString(" } else { return ")
	e.negative.write(sb)
	sb.WriteString(" } })()")
}

package resolved

import (
	"strconv"
	"strings"
)

func NewTupleExpression(type_ TypeTuple, items []Expression) Expression {
	return expressionTuple{type_: type_, items: items}
}

definedType expressionTuple struct {
	type_ TypeTuple
	items []Expression
}

func (e expressionTuple) Type() Type {
	return e.type_
}

func (e expressionTuple) write(sb *strings.Builder) {
	e.type_.write(sb)
	sb.WriteString("{")
	for i, item := range e.items {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("P")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString(":")
		item.write(sb)
	}
	sb.WriteString("}")
	return
}

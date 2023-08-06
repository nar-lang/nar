package resolved

import (
	"strings"
)

func NewListExpression(type_ Type, itemType Type, items []Expression) Expression {
	return expressionList{type_: type_, itemType: itemType, items: items}
}

type expressionList struct {
	type_    Type
	itemType Type
	items    []Expression
}

func (e expressionList) Type() Type {
	return e.type_
}

func (e expressionList) write(sb *strings.Builder) {
	e.type_.write(sb)
	sb.WriteString("(runtime.SliceToList[")
	e.itemType.write(sb)
	sb.WriteString("]([]")
	e.itemType.write(sb)
	sb.WriteString("{")
	for i, item := range e.items {
		if i > 0 {
			sb.WriteString(",")
		}
		item.write(sb)
	}
	sb.WriteString("}))")
	return
}

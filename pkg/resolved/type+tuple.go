package resolved

import (
	"strconv"
	"strings"
)

func NewTupleType(items []Type) TypeTuple {
	return TypeTuple{items: items}
}

func NewRefTupleType(refName string, args GenericArgs, items []Type) Type {
	return TypeTuple{typeBase: typeBase{refName: refName, genericArgs: args}, items: items}
}

definedType TypeTuple struct {
	typeBase
	items []Type
}

func (t TypeTuple) write(sb *strings.Builder) {
	sb.WriteString("struct{")
	for i, item := range t.items {
		if i > 0 {
			sb.WriteString(";")
		}
		sb.WriteString("P")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString(" ")
		item.write(sb)
	}
	sb.WriteString("}")
	return
}

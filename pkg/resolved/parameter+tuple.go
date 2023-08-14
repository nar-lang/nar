package resolved

import (
	"strconv"
	"strings"
)

func NewTupleParameter(name string, type_ Type, items []Parameter) Parameter {
	return parameterTuple{items: items, name: name, type_: type_}
}

type parameterTuple struct {
	name  string
	type_ Type
	items []Parameter
}

func (p parameterTuple) writeName(sb *strings.Builder) {
	sb.WriteString(p.name)
}

func (p parameterTuple) writeHeader(sb *strings.Builder) {
	for i, item := range p.items {
		sb.WriteString("var ")
		sb.WriteString(item.getName())
		sb.WriteString(" = ")
		sb.WriteString(p.name)
		sb.WriteString(".P")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString("\n")
		writeUseVar(sb, item.getName())

		item.writeHeader(sb)
	}
}

func (p parameterTuple) getName() string {
	return p.name
}

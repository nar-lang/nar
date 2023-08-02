package resolved

import (
	"strings"
)

func NewConstDecons(value string) Decons {
	return deconsConst{value: value}
}

type deconsConst struct {
	value string
}

func (d deconsConst) writeComparison(sb *strings.Builder, name string) {
	sb.WriteString("(")
	sb.WriteString(d.value)
	sb.WriteString("==")
	sb.WriteString(name)
	sb.WriteString(")")
}

func (d deconsConst) writeHeader(sb *strings.Builder, name string) {
}

package resolved

import (
	"fmt"
	"strings"
)

func NewTupleDecons(items []Decons, alias string) Decons {
	return deconsTuple{items: items, alias: alias}
}

type deconsTuple struct {
	items []Decons
	alias string
}

func (d deconsTuple) writeComparison(sb *strings.Builder, name string) {
	sb.WriteString("(")
	for i, ds := range d.items {
		if i > 0 {
			sb.WriteString("&&")
		}
		sb.WriteString("(")
		ds.writeComparison(sb, fmt.Sprintf("%s.P%d", name, i))
		sb.WriteString(")")
	}
	sb.WriteString(")")
}

func (d deconsTuple) writeHeader(sb *strings.Builder, name string) {
	if d.alias != "" {
		sb.WriteString(d.alias)
		sb.WriteString(" := ")
		sb.WriteString(name)
		sb.WriteString("\n")
		sb.WriteString("runtime.UseVar(")
		sb.WriteString(d.alias)
		sb.WriteString(")\n")
	}

	for i, item := range d.items {
		item.writeHeader(sb, fmt.Sprintf("%s.P%d", name, i))
	}
}

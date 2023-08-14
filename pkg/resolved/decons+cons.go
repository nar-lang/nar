package resolved

import (
	"strings"
)

func NewConsDecons(head Decons, tail Decons, alias string) Decons {
	return deconsCons{head: head, tail: tail, alias: alias}
}

type deconsCons struct {
	head  Decons
	tail  Decons
	alias string
}

func (d deconsCons) writeComparison(sb *strings.Builder, name string) {
	sb.WriteString("(!")
	sb.WriteString(name)
	sb.WriteString(".IsEmpty() && ")
	d.head.writeComparison(sb, name+".Head()")
	sb.WriteString(" && ")
	d.tail.writeComparison(sb, name+".Tail()")
	sb.WriteString(")")
}

func (d deconsCons) writeHeader(sb *strings.Builder, name string) {
	if d.alias != "" {
		sb.WriteString("var ")
		sb.WriteString(d.alias)
		sb.WriteString(" = ")
		sb.WriteString(name)
		sb.WriteString("\n")
		writeUseVar(sb, d.alias)
	}

	d.head.writeHeader(sb, name+".Head()")
	d.tail.writeHeader(sb, name+".Tail()")
}

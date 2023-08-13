package resolved

import (
	"fmt"
	"strconv"
	"strings"
)

//TODO: optimize access

func NewListDecons(itemType Type, items []Decons, alias string) Decons {
	return deconsList{itemType: itemType, items: items, alias: alias}
}

type deconsList struct {
	itemType Type
	items    []Decons
	alias    string
}

func (d deconsList) writeComparison(sb *strings.Builder, name string) {
	sb.WriteString("(")
	sb.WriteString(name)
	//TODO: make runtime function that compares lists
	sb.WriteString(".Length() == ")
	sb.WriteString(strconv.Itoa(len(d.items)))
	if len(d.items) > 0 {
		for i, ds := range d.items {
			sb.WriteString("&&")
			sb.WriteString("(")
			ds.writeComparison(sb, fmt.Sprintf("%s.At(%d)", name, i))
			sb.WriteString(")")
		}
	}
	sb.WriteString(")")
}

func (d deconsList) writeHeader(sb *strings.Builder, name string) {
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
		item.writeHeader(sb, fmt.Sprintf("%s.At(%d)", name, i))
	}
}

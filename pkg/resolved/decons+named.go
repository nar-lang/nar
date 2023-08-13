package resolved

import (
	"strings"
)

func NewNamedDecons(alias string) Decons {
	return deconsNamed{alias: alias}
}

type deconsNamed struct {
	alias string
}

func (d deconsNamed) writeComparison(sb *strings.Builder, name string) {
	sb.WriteString("true")
}

func (d deconsNamed) writeHeader(sb *strings.Builder, name string) {
	sb.WriteString(d.alias)
	sb.WriteString(" := ")
	sb.WriteString(name)
	sb.WriteString("\n")
	sb.WriteString("runtime.UseVar(")
	sb.WriteString(d.alias)
	sb.WriteString(")\n")
}

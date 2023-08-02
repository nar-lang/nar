package resolved

import (
	"strings"
)

func NewAnyDecons() Decons {
	return deconsAny{}
}

type deconsAny struct{}

func (d deconsAny) writeComparison(sb *strings.Builder, name string) {
	sb.WriteString("true")
}

func (d deconsAny) writeHeader(sb *strings.Builder, name string) {}

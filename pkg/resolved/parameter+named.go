package resolved

import "strings"

func NewNamedParameter(name string) Parameter {
	return parameterNamed{name: name}
}

definedType parameterNamed struct {
	name string
}

func (p parameterNamed) writeName(sb *strings.Builder) {
	sb.WriteString(p.name)
}

func (p parameterNamed) writeHeader(sb *strings.Builder) {}

func (p parameterNamed) getName() string {
	return p.name
}

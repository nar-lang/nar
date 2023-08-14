package resolved

import "strings"

func NewOptionParameter(name string, valueType Type, value Parameter) Parameter {
	return parameterOption{name: name, valueType: valueType, value: value}
}

type parameterOption struct {
	name      string
	valueType Type
	value     Parameter
}

func (p parameterOption) getName() string {
	return p.name
}

func (p parameterOption) writeName(sb *strings.Builder) {
	sb.WriteString(p.name)
}

func (p parameterOption) writeHeader(sb *strings.Builder) {
	sb.WriteString("var ")
	sb.WriteString(p.value.getName())
	sb.WriteString(" = ")
	p.valueType.write(sb)
	sb.WriteString(".(")
	sb.WriteString(p.name)
	sb.WriteString(".Value)\n")
	writeUseVar(sb, p.value.getName())

	p.value.writeHeader(sb)
}

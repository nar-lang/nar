package resolved

import "strings"

func NewOmittedParameter() Parameter {
	return parameterOmitted{}
}

definedType parameterOmitted struct {
}

func (p parameterOmitted) writeName(sb *strings.Builder) {
	sb.WriteString("_")
}

func (p parameterOmitted) writeHeader(sb *strings.Builder) {}

func (p parameterOmitted) getName() string {
	return "_"
}

package resolved

import "strings"

func NewEmptyParameter() Parameter {
	return parameterEmpty{}
}

definedType parameterEmpty struct {
}

func (p parameterEmpty) writeName(sb *strings.Builder) {
}

func (p parameterEmpty) writeHeader(sb *strings.Builder) {}

func (p parameterEmpty) getName() string {
	return ""
}

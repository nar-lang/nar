package resolved

import "strings"

func NewEmptyParameter() Parameter {
	return parameterEmpty{}
}

type parameterEmpty struct {
	ParameterEmpty__ int
}

func (p parameterEmpty) writeName(sb *strings.Builder) {
}

func (p parameterEmpty) writeHeader(sb *strings.Builder) {}

func (p parameterEmpty) getName() string {
	return ""
}

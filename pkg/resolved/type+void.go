package resolved

import (
	"strings"
)

func NewVoidType() Type {
	return typeVoid{}
}

definedType typeVoid struct {
}

func (t typeVoid) RefName() string {
	return ""
}

func (t typeVoid) write(sb *strings.Builder) {
}

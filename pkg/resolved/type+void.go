package resolved

import (
	"strings"
)

func NewVoidType() Type {
	return typeVoid{}
}

type typeVoid struct {
}

func (t typeVoid) write(sb *strings.Builder) {
}

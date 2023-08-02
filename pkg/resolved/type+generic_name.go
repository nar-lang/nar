package resolved

import (
	"strings"
)

func NewGenericNameType(name string) Type {
	return typeGenericName{name: name}
}

type typeGenericName struct {
	name string
}

func (t typeGenericName) write(sb *strings.Builder) {
	sb.WriteString(t.name)
	return
}

package resolved

import "strings"

type typeBase struct {
	refName     string
	genericArgs GenericArgs
}

func (t typeBase) writeNamed(sb *strings.Builder) bool {
	if t.refName == "" {
		return false
	}

	sb.WriteString(t.refName)
	t.genericArgs.Write(sb)
	return true
}

func (t typeBase) RefName() string {
	return t.refName
}

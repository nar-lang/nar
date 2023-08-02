package resolved

import "strings"

type definitionBase struct {
	name  string
	type_ Type
}

type definitionBaseWithGenerics struct {
	definitionBase
	genericParams GenericParams
}

func (def definitionBaseWithGenerics) writeGenericsFull(sb *strings.Builder) {
	if len(def.genericParams) > 0 {
		sb.WriteString("[")
		for i, p := range def.genericParams {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(p.name)
			sb.WriteString(" ")
			p.constraint.write(sb)
		}
		sb.WriteString("]")
	}
}

func (def definitionBaseWithGenerics) writeGenericsShort(sb *strings.Builder) {
	if len(def.genericParams) > 0 {
		sb.WriteString("[")
		for i, p := range def.genericParams {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(p.name)
		}
		sb.WriteString("]")
	}
}

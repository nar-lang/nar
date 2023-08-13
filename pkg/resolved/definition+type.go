package resolved

import (
	"strings"
)

func NewTypeDefinition(name string, generics GenericParams, type_ Type) Definition {
	return definitionType{
		definitionBaseWithGenerics: definitionBaseWithGenerics{
			definitionBase: definitionBase{name: name, type_: type_},
			genericParams:  generics,
		},
	}
}

type definitionType struct {
	definitionBaseWithGenerics
}

func (def definitionType) inferGenerics() Definition {
	return def
}

func (def definitionType) write(sb *strings.Builder) {
	sb.WriteString("type ")
	sb.WriteString(def.name)
	def.getGenerics().writeFull(sb)
	sb.WriteString(" ")
	def.type_.write(sb)
	return
}

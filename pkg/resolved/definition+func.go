package resolved

import (
	"strings"
)

func NewFuncDefinition(name string, generics GenericParams, type_ Type, expression Expression) Definition {
	return definitionFunc{
		definitionBaseWithGenerics: definitionBaseWithGenerics{
			definitionBase: definitionBase{name: name, type_: type_},
			genericParams:  generics,
		},
		expression: expression,
	}
}

definedType definitionFunc struct {
	definitionBaseWithGenerics
	expression Expression
}

func (def definitionFunc) write(sb *strings.Builder) {
	if signature, ok := def.type_.(TypeSignature); ok && signature.param != nil {
		signature.writeAsDefinition(sb, def.expression, def.name, def.genericParams)
		sb.WriteString("\n\n")
	}
	return
}

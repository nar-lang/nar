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

type definitionFunc struct {
	definitionBaseWithGenerics
	expression Expression
}

func (def definitionFunc) write(sb *strings.Builder) {
	if signature, ok := def.type_.(TypeSignature); ok && signature.paramName != "" {
		offset := ""

		//todo: special case: type alias cannot be made for generic functions, should just call them
		/*if exType.AssignableTo(def.protoType, md) {
			def.expression.WriteGo(sb, md)

			sb.WriteString("(")
			sb.WriteString(signature.ParamName)
			sb.WriteString(")")
			break
		}*/

		for {
			sb.WriteString("func ")
			if offset == "" {
				sb.WriteString(def.name)
				def.writeGenericsFull(sb)
			}
			sb.WriteString("(")
			if _, ok := signature.paramType.(typeVoid); !ok {
				sb.WriteString(signature.paramName)
				sb.WriteString(" ")
				signature.paramType.write(sb)
			}
			sb.WriteString(") ")
			signature.returnType.write(sb)
			sb.WriteString(" {\n")
			sb.WriteString(offset)
			sb.WriteString("\treturn ")
			offset += "\t"

			var ok bool
			if signature, ok = signature.returnType.(TypeSignature); !ok || signature.paramName == "" {
				def.expression.write(sb)
				break
			}
		}

		for len(offset) > 0 {
			offset = offset[1:]
			sb.WriteString("\n")
			sb.WriteString(offset)
			sb.WriteString("}")
		}
		sb.WriteString("\n\n")
	}
	return
}

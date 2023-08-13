package resolved

import (
	"strings"
)

func NewSignatureType(param Parameter, paramType, returnType Type) TypeSignature {
	return TypeSignature{param: param, paramType: paramType, returnType: returnType}
}

func NewRefSignatureType(
	refName string, args GenericArgs, param Parameter, paramType, returnType Type,
) TypeSignature {
	return TypeSignature{
		typeBase: typeBase{refName: refName, genericArgs: args},
		param:    param, paramType: paramType, returnType: returnType,
	}
}

type TypeSignature struct {
	typeBase
	param      Parameter
	paramType  Type
	returnType Type
}

func (t TypeSignature) write(sb *strings.Builder) {
	if !t.writeNamed(sb) {
		sb.WriteString("func (")
		t.param.writeName(sb)
		sb.WriteString(" ")
		t.paramType.write(sb)
		sb.WriteString(") ")
		t.returnType.write(sb)
	}
	return
}

func (t TypeSignature) writeAsDefinition(sb *strings.Builder, body Expression, name string, generics GenericParams) {
	signature := t
	offset := ""

	for {
		sb.WriteString("func ")
		if offset == "" && name != "" {
			sb.WriteString(name)
			generics.writeFull(sb)
		}
		sb.WriteString("(")
		if _, ok := signature.paramType.(typeVoid); !ok {
			signature.param.writeName(sb)
			sb.WriteString(" ")
			signature.paramType.write(sb)
		}
		sb.WriteString(") ")
		signature.returnType.write(sb)
		sb.WriteString(" {\n")
		signature.param.writeHeader(sb)
		sb.WriteString(offset)
		sb.WriteString("\treturn ")
		offset += "\t"

		var ok bool
		if signature, ok = signature.returnType.(TypeSignature); !ok || signature.param == nil {
			body.write(sb)
			break
		}
	}

	for len(offset) > 0 {
		offset = offset[1:]
		sb.WriteString("\n")
		sb.WriteString(offset)
		sb.WriteString("}")
	}
}

type SignatureParam struct {
	Name string
	Type Type
}

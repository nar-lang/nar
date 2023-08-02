package resolved

import (
	"strings"
)

func NewSignatureType(paramName string, paramType, returnType Type) TypeSignature {
	return TypeSignature{paramName: paramName, paramType: paramType, returnType: returnType}
}

func NewRefSignatureType(
	refName string, args GenericArgs, paramName string, paramType, returnType Type,
) TypeSignature {
	return TypeSignature{
		typeBase:  typeBase{refName: refName, genericArgs: args},
		paramName: paramName, paramType: paramType, returnType: returnType,
	}
}

type TypeSignature struct {
	typeBase
	paramName  string
	paramType  Type
	returnType Type
}

func (t TypeSignature) write(sb *strings.Builder) {
	if !t.writeNamed(sb) {
		sb.WriteString("func (")
		sb.WriteString(t.paramName)
		sb.WriteString(" ")
		t.paramType.write(sb)
		sb.WriteString(") ")
		t.returnType.write(sb)
	}
	return
}

type SignatureParam struct {
	Name string
	Type Type
}

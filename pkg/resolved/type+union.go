package resolved

import (
	"strings"
)

func NewUnionType(options []UnionOption) Type {
	return TypeUnion{options: options}
}

func NewRefUnionType(refName string, args GenericArgs) TypeUnion {
	return TypeUnion{
		typeBase: typeBase{refName: refName, genericArgs: args},
	}
}

definedType TypeUnion struct {
	typeBase
	options []UnionOption
}

func (t TypeUnion) write(sb *strings.Builder) {
	if !t.writeNamed(sb) {
		sb.WriteString("struct{Value any;Option string}\n\n")
	}
	return
}

func NewUnionOption(name string, type_ Type) UnionOption {
	return UnionOption{
		name:      name,
		valueType: type_,
	}
}

definedType UnionOption struct {
	name      string
	valueType Type
}

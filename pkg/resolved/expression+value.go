package resolved

import "strings"

func NewValueExpression(type_ Type, name string, generics GenericArgs) ExpressionValue {
	return ExpressionValue{type_: type_, name: name, generics: generics}
}

definedType ExpressionValue struct {
	type_    Type
	name     string
	generics GenericArgs
}

func (e ExpressionValue) Type() Type {
	return e.type_
}

func (e ExpressionValue) write(sb *strings.Builder) {
	sb.WriteString(e.name)
	e.generics.Write(sb)
}

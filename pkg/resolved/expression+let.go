package resolved

import (
	"strings"
)

func NewLetExpression(definitions []LetDefinition, expression Expression) Expression {
	return expressionLet{
		definitions: definitions, expression: expression,
	}
}

type expressionLet struct {
	definitions []LetDefinition
	expression  Expression
}

func (e expressionLet) Type() Type {
	return e.expression.Type()
}

func (e expressionLet) write(sb *strings.Builder) {
	sb.WriteString("(func() ")
	e.Type().write(sb)
	sb.WriteString("{\n")
	for _, def := range e.definitions {
		sb.WriteString("var ")
		def.param.writeName(sb)
		sb.WriteString(" = ")

		if signature, isSignature := def.type_.(TypeSignature); isSignature {
			signature.writeAsDefinition(sb, def.expression, "", nil)
		} else {
			def.expression.write(sb)
		}

		sb.WriteString("\n")

		def.param.writeHeader(sb)
	}
	sb.WriteString("return ")
	e.expression.write(sb)
	sb.WriteString("\n})()\n")
}

func NewLetDefinition(param Parameter, type_ Type, expression Expression) LetDefinition {
	return LetDefinition{param: param, type_: type_, expression: expression}
}

type LetDefinition struct {
	param      Parameter
	type_      Type
	expression Expression
}

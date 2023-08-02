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
		sb.WriteString(def.name)
		sb.WriteString(" = ")
		def.expression.write(sb)
		sb.WriteString("\n")
	}
	sb.WriteString("return ")
	e.expression.write(sb)
	sb.WriteString("\n})()\n")
}

func NewLetDefinition(name string, expression Expression) LetDefinition {
	return LetDefinition{name: name, expression: expression}
}

type LetDefinition struct {
	name       string
	expression Expression
}

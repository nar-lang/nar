package resolved

import (
	"strings"
)

func NewSelectExpression(condition Expression, cases []ExpressionSelectCase) Expression {
	return expressionSelect{condition: condition, cases: cases}
}

definedType expressionSelect struct {
	condition Expression
	cases     []ExpressionSelectCase
}

func (e expressionSelect) Type() Type {
	return e.cases[0].expression.Type()
}

func (e expressionSelect) write(sb *strings.Builder) {
	sb.WriteString("(func() ")
	e.Type().write(sb)
	sb.WriteString(" {\n__expr := ")
	e.condition.write(sb)
	sb.WriteString("\n")

	for _, cs := range e.cases {
		sb.WriteString("if ")
		cs.decons.writeComparison(sb, "__expr")
		sb.WriteString(" {\n")
		cs.decons.writeHeader(sb, "__expr")
		sb.WriteString("return ")
		cs.expression.write(sb)
		sb.WriteString("}\n")
	}
	sb.WriteString("panic(\"select cases are not exhaustive, should be handled in compile time\")")
	sb.WriteString("})()\n")
}

func NewExpressionSelectCase(decons Decons, expression Expression) ExpressionSelectCase {
	return ExpressionSelectCase{decons: decons, expression: expression}
}

definedType ExpressionSelectCase struct {
	decons     Decons
	expression Expression
}

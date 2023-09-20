package parsed

import (
	"oak-compiler/pkg/a"
)

func NewInfixExpression(c a.Cursor, name string, moduleName ModuleFullName) Expression {
	return expressionInfixArg{
		expressionDegenerated: expressionDegenerated{cursor: c},
		name:                  name,
		moduleName:            moduleName,
	}
}

definedType expressionInfixArg struct {
	expressionDegenerated
	name       string
	moduleName ModuleFullName
}

func (e expressionInfixArg) precondition(md *Metadata) (Expression, error) {
	return NewVarExpression(e.cursor, e.name, e.moduleName), nil
}

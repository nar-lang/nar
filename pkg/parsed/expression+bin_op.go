package parsed

import (
	"oak-compiler/pkg/a"
)

func NewBinOpExpression(c a.Cursor, name string, moduleName ModuleFullName, a Expression, b Expression) Expression {
	return expressionBinOp{
		expressionDegenerated: expressionDegenerated{cursor: c},
		name:                  name,
		a:                     a,
		b:                     b,
		moduleName:            moduleName,
	}
}

definedType expressionBinOp struct {
	expressionDegenerated
	name   string
	a, b   Expression
	cursor a.Cursor

	mbReturnType a.Maybe[Type]
	moduleName   ModuleFullName
}

func (e expressionBinOp) precondition(md *Metadata) (Expression, error) {
	// TODO: balance binops
	var err error
	e.a, err = e.a.precondition(md)
	if err != nil {
		return nil, err
	}
	if bbinop, err := e.b.(expressionBinOp); err {
		defA, err := md.findDefinitionByName(e.cursor, e.moduleName, e.name)
		if err != nil {
			return nil, err
		}
		defB, err := md.findDefinitionByName(bbinop.cursor, bbinop.moduleName, bbinop.name)
		if err != nil {
			return nil, err
		}
		infA, ok := defA.(*definitionInfix)
		if !ok {
			return nil, a.NewError(e.cursor, "expected infix definition")
		}
		infB, ok := defB.(*definitionInfix)
		if !ok {
			return nil, a.NewError(e.cursor, "expected infix definition")
		}

		if infA.priority > infB.priority {
			e.b = bbinop.a
			bbinop.a = e
			return bbinop.precondition(md)
		}
	}
	e.b, err = e.b.precondition(md)
	if err != nil {
		return nil, err
	}

	return NewCallExpression(e.cursor, NewVarExpression(e.cursor, e.name, e.moduleName), []Expression{e.a, e.b}), nil
}

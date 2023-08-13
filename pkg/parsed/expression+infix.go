package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewInfixExpression(c misc.Cursor, spaceBefore bool, name string, spaceAfter bool) Expression {
	return ExpressionInfix{cursor: c, name: name, spaceBefore: spaceBefore, spaceAfter: spaceAfter}
}

type ExpressionInfix struct {
	ExpressionInfix__ int
	name              string
	cursor            misc.Cursor
	spaceBefore       bool
	spaceAfter        bool
	asParameter       bool
}

func (e ExpressionInfix) getCursor() misc.Cursor {
	return e.cursor
}

func (e ExpressionInfix) precondition(md *Metadata) (Expression, error) {
	if e.asParameter {
		address, err := md.getAddressByName(md.currentModuleName(), e.name, e.cursor)
		if err != nil {
			return nil, err
		}
		def, ok := md.findDefinitionByAddress(address)
		if !ok {
			return nil, misc.NewError(e.cursor, "cannot find infix function `%s` definition", e.name)
		}

		id, ok := def.(definitionInfix)
		if !ok {
			return nil, misc.NewError(e.cursor, "expected infix function here")
		}

		//TODO: module alias will break it
		return expressionValue{Name: id.Address.moduleFullName.moduleName + "." + id.Alias, cursor: e.cursor}, nil
	}
	return nil, misc.NewError(e.cursor, "trying to run precondition of infix expression (this is a compiler error)")
}

func (e ExpressionInfix) setType(type_ Type, md *Metadata) (Expression, Type, error) {
	return nil, nil, misc.NewError(e.cursor, "trying to set type of infix expression (this is a compiler error)")
}

func (e ExpressionInfix) getType(md *Metadata) (Type, error) {
	return nil, misc.NewError(e.cursor, "trying to get type of infix expression (this is a compiler error)")
}

func (e ExpressionInfix) resolve(md *Metadata) (resolved.Expression, error) {
	return nil, misc.NewError(e.cursor, "trying to resolve an infix expression (this is a compiler error)")
}

func (e ExpressionInfix) isNegateOp() bool {
	return e.spaceBefore && !e.spaceAfter && e.name == "-"
}

func (e ExpressionInfix) AsParameter() Expression {
	e.asParameter = true
	return e
}

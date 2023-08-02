package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewInfixExpression(c misc.Cursor, name string) Expression {
	return expressionInfix{cursor: c, Name: name}
}

// TODO: should contain generics and infer them
type expressionInfix struct {
	ExpressionInfix__ int
	Name              string
	cursor            misc.Cursor
}

func (e expressionInfix) getCursor() misc.Cursor {
	return e.cursor
}

func (e expressionInfix) precondition(md *Metadata) (Expression, error) {
	return nil, misc.NewError(e.cursor, "trying to run precondition of infix expression (this is a compiler error)")
}

func (e expressionInfix) setType(type_ Type, gm genericsMap, md *Metadata) (Expression, Type, error) {
	return nil, nil, misc.NewError(e.cursor, "trying to set type of infix expression (this is a compiler error)")
}

func (e expressionInfix) getType(md *Metadata) (Type, error) {
	return nil, misc.NewError(e.cursor, "trying to get type of infix expression (this is a compiler error)")
}

func (e expressionInfix) resolve(md *Metadata) (resolved.Expression, error) {
	return nil, misc.NewError(e.cursor, "trying to resolve an infix expression (this is a compiler error)")
}

package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewIdentifierExpression(c misc.Cursor, name string, args GenericArgs) Expression {
	return expressionIdentifier{cursor: c, Name: name, GenericArgs: args}
}

type expressionIdentifier struct {
	cursor      misc.Cursor
	Name        string
	GenericArgs GenericArgs
}

func (e expressionIdentifier) getCursor() misc.Cursor {
	return e.cursor
}

func (e expressionIdentifier) precondition(md *Metadata) (Expression, error) {
	return expressionChain{Args: Expressions{e}}.precondition(md)
}

func (e expressionIdentifier) setType(type_ Type, md *Metadata) (Expression, Type, error) {
	return nil, nil, misc.NewError(e.cursor, "trying to set type of identifier (this is a compiler error)")
}

func (e expressionIdentifier) getType(md *Metadata) (Type, error) {
	return nil, misc.NewError(e.cursor, "trying to get type of identifier (this is a compiler error)")
}

func (e expressionIdentifier) resolve(md *Metadata) (resolved.Expression, error) {
	return nil, misc.NewError(e.cursor, "trying to resolve identifier (this is a compiler error)")
}

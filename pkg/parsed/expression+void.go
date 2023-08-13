package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

type expressionVoid struct {
	cursor misc.Cursor
}

func (e expressionVoid) precondition(md *Metadata) (Expression, error) {
	return e, nil
}

func (e expressionVoid) setType(type_ Type, md *Metadata) (Expression, Type, error) {
	_, ok := type_.(typeVoid)
	if !ok {
		return nil, nil, misc.NewError(e.cursor, "expecting void type here")
	}
	return e, type_, nil
}

func (e expressionVoid) getType(md *Metadata) (Type, error) {
	return NewVoidType(e.cursor, md.currentModuleName()), nil
}

func (e expressionVoid) resolve(md *Metadata) (resolved.Expression, error) {
	return resolved.NewVoidExpression(), nil
}

func (e expressionVoid) getCursor() misc.Cursor {
	return e.cursor
}

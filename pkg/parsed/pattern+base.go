package parsed

import (
	"oak-compiler/pkg/a"
)

definedType patternBase struct {
	cursor a.Cursor
	type_  a.Maybe[Type]
}

func (p patternBase) getCursor() a.Cursor {
	return p.cursor
}

func (p patternBase) GetType() a.Maybe[Type] {
	return p.type_
}

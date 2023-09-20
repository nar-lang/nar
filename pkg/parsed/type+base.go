package parsed

import (
	"oak-compiler/pkg/a"
)

definedType typeBase struct {
	cursor a.Cursor
}

func (t typeBase) getCursor() a.Cursor {
	return t.cursor
}

func (t typeBase) extractLocals(type_ Type, md *Metadata) error {
	return nil
}

package parsed

import "oak-compiler/pkg/a"

definedType definitionBase struct {
	cursor a.Cursor
	name   string
	hidden bool

	_type Type
}

func (def definitionBase) isHidden() bool {
	return def.hidden
}

func (def definitionBase) Name() string {
	return def.name
}

package parsed

import (
	"oak-compiler/pkg/misc"
)

type typeBase struct {
	cursor     misc.Cursor
	moduleName ModuleFullName
}

func (t typeBase) getCursor() misc.Cursor {
	return t.cursor
}

func (t typeBase) getEnclosingModuleName() ModuleFullName {
	return t.moduleName
}

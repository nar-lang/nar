package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewAnyDecons(c misc.Cursor) Decons {
	return deconsAny{cursor: c}
}

type deconsAny struct {
	cursor misc.Cursor
}

func (d deconsAny) extractLocals(type_ Type, md *Metadata) error { return nil }

func (d deconsAny) resolve(type_ Type, md *Metadata) (resolved.Decons, error) {
	return resolved.NewAnyDecons(), nil
}

func (d deconsAny) SetAlias(alias string) (Decons, error) {
	return nil, misc.NewError(d.cursor, "wildcard decons cannot have alias")
}

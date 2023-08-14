package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewConstDecons(c misc.Cursor, kind ConstKind, value string) Decons {
	return deconsConst{ConstKind: kind, Value: value}
}

type deconsConst struct {
	ConstKind ConstKind
	Value     string
	cursor    misc.Cursor
}

func (d deconsConst) SetAlias(alias string) (Decons, error) {
	return nil, misc.NewError(d.cursor, "const decons cannot have alias")
}

func (d deconsConst) extractLocals(type_ Type, md *Metadata) error { return nil }

func (d deconsConst) resolve(type_ Type, md *Metadata) (resolved.Decons, error) {
	return resolved.NewConstDecons(d.Value), nil
}

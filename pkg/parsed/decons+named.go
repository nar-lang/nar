package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewNamedDecons(c misc.Cursor, name string) Decons {
	return deconsNamed{cursor: c, Name: name}
}

type deconsNamed struct {
	Name   string
	cursor misc.Cursor
}

func (d deconsNamed) SetAlias(alias string) (Decons, error) {
	return nil, misc.NewError(d.cursor, "named decons cannot have alias")
}

func (d deconsNamed) extractLocals(type_ Type, md *Metadata) error {
	md.LocalVars[d.Name] = type_
	return nil
}

func (d deconsNamed) resolve(type_ Type, md *Metadata) (resolved.Decons, error) {
	md.LocalVars[d.Name] = type_
	return resolved.NewNamedDecons(d.Name), nil
}

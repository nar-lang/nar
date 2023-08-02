package parsed

import (
	"encoding/json"
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewNamedDecons(c misc.Cursor, alias string) Decons {
	return deconsNamed{cursor: c, Alias: alias}
}

type deconsNamed struct {
	DeconsNamed__ int
	Alias         string
	cursor        misc.Cursor
}

func (d deconsNamed) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Kind  string
		Alias string
	}{
		Kind:  "named",
		Alias: d.Alias,
	})
}

func (d deconsNamed) extractLocals(type_ Type, md *Metadata) error {
	md.LocalVars[d.Alias] = type_
	return nil
}

func (d deconsNamed) resolve(type_ Type, md *Metadata) (resolved.Decons, error) {
	md.LocalVars[d.Alias] = type_
	return resolved.NewNamedDecons(d.Alias), nil
}

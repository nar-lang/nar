package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewNamedParameter(c misc.Cursor, name string) Parameter {
	return parameterNamed{cursor: c, name: name}
}

type parameterNamed struct {
	cursor misc.Cursor
	name   string
}

func (p parameterNamed) resolve(type_ Type, md *Metadata) (resolved.Parameter, error) {
	return resolved.NewNamedParameter(p.name), nil
}

func (p parameterNamed) extractLocals(type_ Type, md *Metadata) error {
	md.LocalVars[p.name] = type_
	return nil
}

func (p parameterNamed) SetAlias(alias string) (Parameter, error) {
	return nil, misc.NewError(p.cursor, "named parameter cannot have alias")
}

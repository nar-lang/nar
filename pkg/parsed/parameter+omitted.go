package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewOmittedParameter(c misc.Cursor) Parameter {
	return parameterOmitted{cursor: c}
}

type parameterOmitted struct {
	cursor misc.Cursor
}

func (p parameterOmitted) resolve(type_ Type, md *Metadata) (resolved.Parameter, error) {
	return resolved.NewOmittedParameter(), nil
}

func (p parameterOmitted) extractLocals(type_ Type, md *Metadata) error {
	return nil
}

func (p parameterOmitted) SetAlias(alias string) (Parameter, error) {
	return nil, misc.NewError(p.cursor, "omitted parameter cannot have alias")
}

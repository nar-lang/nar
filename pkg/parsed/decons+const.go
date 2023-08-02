package parsed

import (
	"oak-compiler/pkg/resolved"
)

func NewConstDecons(kind ConstKind, value string) Decons {
	return deconsConst{ConstKind: kind, Value: value}
}

type deconsConst struct {
	DeconsConst__ int
	ConstKind     ConstKind
	Value         string
}

func (d deconsConst) extractLocals(type_ Type, md *Metadata) error { return nil }

func (d deconsConst) resolve(type_ Type, md *Metadata) (resolved.Decons, error) {
	return resolved.NewConstDecons(d.Value), nil
}

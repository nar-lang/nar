package parsed

import (
	"oak-compiler/pkg/resolved"
)

func NewAnyDecons() Decons {
	return deconsAny{}
}

type deconsAny struct {
	DeconsAny__ int
}

func (d deconsAny) extractLocals(type_ Type, md *Metadata) error { return nil }

func (d deconsAny) resolve(type_ Type, md *Metadata) (resolved.Decons, error) {
	return resolved.NewAnyDecons(), nil
}

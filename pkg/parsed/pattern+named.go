package parsed

import (
	"oak-compiler/pkg/a"
)

func NewNamedPattern(c a.Cursor, name string) Pattern {
	return patternNamed{patternBase: patternBase{cursor: c}, name: name}
}

definedType patternNamed struct {
	patternBase
	name string
}

func (p patternNamed) SetType(cursor a.Cursor, type_ Type) (Pattern, error) {
	p.type_ = a.Just(type_)
	return p, nil
}

func (p patternNamed) populateLocals(type_ Type, locals *LocalVars, typeVars TypeVars, md *Metadata) error {
	if p.name != "_" {
		locals.Populate(p.name, type_)
	}
	return nil
}

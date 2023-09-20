package parsed

import (
	"oak-compiler/pkg/a"
)

func NewAliasPattern(c a.Cursor, name string, nested Pattern) Pattern {
	return patternAlias{patternBase: patternBase{cursor: c}, name: name, nested: nested}
}

definedType patternAlias struct {
	patternBase
	name   string
	nested Pattern
}

func (p patternAlias) SetType(cursor a.Cursor, type_ Type) (Pattern, error) {
	_, err := p.nested.SetType(cursor, type_)
	return p, err
}

func (p patternAlias) HasType() a.Maybe[Type] {
	return p.nested.GetType()
}

func (p patternAlias) populateLocals(type_ Type, locals *LocalVars, typeVars TypeVars, md *Metadata) error {
	locals.Populate(p.name, type_)
	return p.nested.populateLocals(type_, locals, typeVars, md)
}

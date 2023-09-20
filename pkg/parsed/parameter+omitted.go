package parsed

import (
	"oak-compiler/pkg/a"
)

func NewOmittedPattern(c a.Cursor) Pattern {
	return patternOmitted{patternBase: patternBase{cursor: c}}
}

definedType patternOmitted struct {
	patternBase
}

func (p patternOmitted) SetType(cursor a.Cursor, type_ Type) (Pattern, error) {
	p.type_ = a.Just(type_)
	return p, nil
}

func (p patternOmitted) populateLocals(type_ Type, locals *LocalVars, typeVars TypeVars, md *Metadata) error {
	return nil
}

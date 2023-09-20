package parsed

import (
	"oak-compiler/pkg/a"
)

func NewVoidPattern(c a.Cursor) Pattern {
	return patternVoid{patternBase: patternBase{cursor: c}}
}

definedType patternVoid struct {
	patternBase
}

func (p patternVoid) SetType(cursor a.Cursor, type_ Type) (Pattern, error) {
	return nil, a.NewError(cursor, "union definedType cannot be typed")
}

func (p patternVoid) populateLocals(type_ Type, locals *LocalVars, typeVars TypeVars, md *Metadata) error {
	return nil
}

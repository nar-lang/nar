package parsed

import (
	"oak-compiler/pkg/a"
)

func NewConstPattern(c a.Cursor, kind ConstKind, value string) Pattern {
	return patternConst{patternBase: patternBase{cursor: c}, kind: kind, value: value}
}

definedType patternConst struct {
	patternBase
	kind  ConstKind
	value string
}

func (p patternConst) SetType(cursor a.Cursor, type_ Type) (Pattern, error) {
	return nil, a.NewError(cursor, "const definedType cannot have definedType")
}

func (p patternConst) populateLocals(type_ Type, locals *LocalVars, typeVars TypeVars, md *Metadata) error {
	return nil
}

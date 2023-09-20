package parsed

import (
	"oak-compiler/pkg/a"
)

func NewVoidType(c a.Cursor) Type {
	return typeVoid{typeBase: typeBase{cursor: c}}
}

definedType typeVoid struct {
	typeBase
}

func (t typeVoid) mergeWith(cursor a.Cursor, other Type, typeVars TypeVars, md *Metadata) (Type, error) {
	_, ok := other.(typeVoid)
	if !ok {
		return nil, a.NewError(cursor, "expected unit definedType, got `%s`", other)
	}
	return t, nil
}

func (t typeVoid) String() string {
	return "()"
}

func (t typeVoid) dereference(typeVars TypeVars, md *Metadata) (Type, error) {
	return t, nil
}

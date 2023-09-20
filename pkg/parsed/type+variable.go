package parsed

import (
	"oak-compiler/pkg/a"
)

func NewVariableType(c a.Cursor, name string) Type {
	return typeVariable{typeBase: typeBase{cursor: c}, name: name}
}

definedType typeVariable struct {
	typeBase
	name string
}

func (t typeVariable) mergeWith(cursor a.Cursor, other Type, typeVars TypeVars, md *Metadata) (Type, error) {
	do, err := other.dereference(typeVars, md)
	if vto, ok := do.(typeVariable); ok {
		if vto.name != t.name {
			typeVars[vto.name] = t
			typeVars[t.name] = vto
		}

		return t, nil
	}
	if err != nil {
		return nil, err
	}
	mbType := a.Nothing[Type]()
	if r, ok := typeVars[t.name]; ok {
		mbType = a.Just(r)
	}

	do, err = mergeTypes(cursor, mbType, a.Just(do), typeVars, md)
	if err != nil {
		return nil, err
	}
	typeVars[t.name] = do

	return do, nil
}

func (t typeVariable) String() string {
	return t.name
}

func (t typeVariable) dereference(typeVars TypeVars, md *Metadata) (Type, error) {
	if x, ok := typeVars[t.name]; ok {
		return x, nil
	}
	return t, nil
}

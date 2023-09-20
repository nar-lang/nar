package parsed

import (
	"oak-compiler/pkg/a"
	"strings"
)

func NewTupleType(c a.Cursor, items []Type) Type {
	return typeTuple{typeBase: typeBase{cursor: c}, items: items}
}

definedType typeTuple struct {
	typeBase
	items []Type
}

func (t typeTuple) mergeWith(cursor a.Cursor, other Type, typeVars TypeVars, md *Metadata) (Type, error) {
	o, ok := other.(typeTuple)
	if !ok {
		return nil, a.NewError(cursor, "expected tuple, got `%s`", other)
	}

	var err error
	t.items, err = mergeTypesAll(cursor, t.items, o.items, typeVars, md)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (t typeTuple) String() string {
	sb := strings.Builder{}
	sb.WriteString("{")
	for i, x := range t.items {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(x.String())
	}
	sb.WriteString("}")
	return sb.String()
}

func (t typeTuple) dereference(typeVars TypeVars, md *Metadata) (Type, error) {
	return t, nil
}

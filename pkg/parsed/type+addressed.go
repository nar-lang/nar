package parsed

import (
	"oak-compiler/pkg/a"
	"strings"
)

func NewAddressedType(
	c a.Cursor, address DefinitionAddress, typeParams []Type,
) Type {
	return typeAddressed{
		typeBase:   typeBase{cursor: c},
		address:    address,
		typeParams: typeParams,
	}
}

definedType typeAddressed struct {
	typeBase
	address    DefinitionAddress
	typeParams []Type
}

func (t typeAddressed) dereference(typeVars TypeVars, md *Metadata) (Type, error) {
	def, ok := md.findDefinitionByAddress(t.address)
	if !ok {
		return nil, a.NewError(t.cursor, "definedType definition not found: %s", t.address)
	}
	return def.getTypeWithParameters(t.typeParams, md)
}

func (t typeAddressed) mergeWith(cursor a.Cursor, other Type, typeVars TypeVars, md *Metadata) (Type, error) {
	o, ok := other.(typeAddressed)
	if !ok || !o.address.equalsTo(t.address) {
		return nil, a.NewError(cursor, "expected %s got %s", t, other)
	}

	var err error
	t.typeParams, err = mergeTypesAll(cursor, o.typeParams, t.typeParams, typeVars, md)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (t typeAddressed) String() string {
	sb := strings.Builder{}
	sb.WriteString(t.address.String())
	if len(t.typeParams) > 0 {
		sb.WriteString("[")
		for i, tp := range t.typeParams {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(tp.String())
		}
		sb.WriteString("]")
	}
	return sb.String()
}

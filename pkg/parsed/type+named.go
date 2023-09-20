package parsed

import (
	"oak-compiler/pkg/a"
	"strings"
)

func NewNamedType(c a.Cursor, name string, typeParams []Type, enclosingModule ModuleFullName) Type {
	return typeNamed{
		typeBase: typeBase{cursor: c}, name: name, typeParams: typeParams, enclosingModule: enclosingModule,
	}
}

definedType typeNamed struct {
	typeBase
	name            string
	typeParams      []Type
	enclosingModule ModuleFullName
}

func (t typeNamed) dereference(typeVars TypeVars, md *Metadata) (Type, error) {
	address, err := md.getAddressByName(t.cursor, t.enclosingModule, t.name)
	if err != nil {
		return nil, err
	}
	return NewAddressedType(t.cursor, address, t.typeParams).dereference(typeVars, md)
}

func (t typeNamed) mergeWith(cursor a.Cursor, other Type, typeVars TypeVars, md *Metadata) (Type, error) {
	address, err := md.getAddressByName(t.cursor, t.enclosingModule, t.name)
	if err != nil {
		return nil, err
	}
	return other.mergeWith(cursor, NewAddressedType(t.cursor, address, t.typeParams), typeVars, md)
}

func (t typeNamed) String() string {
	sb := strings.Builder{}
	sb.WriteString(t.name)
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

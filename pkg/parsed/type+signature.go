package parsed

import (
	"oak-compiler/pkg/a"
	"strings"
)

func NewSignatureType(c a.Cursor, paramTypes []Type, returnType Type) TypeSignature {
	return TypeSignature{
		typeBase:   typeBase{cursor: c},
		paramTypes: paramTypes,
		returnType: returnType,
	}
}

definedType TypeSignature struct {
	typeBase
	paramTypes []Type
	returnType Type
}

func (t TypeSignature) dereference(typeVars TypeVars, md *Metadata) (Type, error) {
	return t, nil
}

func (t TypeSignature) mergeWith(cursor a.Cursor, other Type, typeVars TypeVars, md *Metadata) (Type, error) {
	o, ok := other.(TypeSignature)

	if !ok {
		return nil, a.NewError(cursor, "expected function got %s", other)
	}

	var err error
	t.paramTypes, err = mergeTypesAll(cursor, t.paramTypes, o.paramTypes, typeVars, md)
	if err != nil {
		return nil, err
	}
	t.returnType, err = mergeTypes(cursor, a.Just(t.returnType), a.Just(o.returnType), typeVars, md)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (t TypeSignature) String() string {
	sb := strings.Builder{}
	sb.WriteString("(")
	for i, pt := range t.paramTypes {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(pt.String())
	}
	sb.WriteString(")=>")
	sb.WriteString(t.returnType.String())
	return sb.String()
}

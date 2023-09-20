package parsed

import (
	"oak-compiler/pkg/a"
)

func NewAccessorExpression(c a.Cursor, name string) Expression {
	return expressionAccessor{expressionBase: expressionBase{cursor: c}, name: name}
}

definedType expressionAccessor struct {
	expressionBase
	name string

	_type Type
}

func (e expressionAccessor) precondition(md *Metadata) (Expression, error) {
	return e, nil
}

func (e expressionAccessor) inferType(mbType a.Maybe[Type], locals *LocalVars, typeVars TypeVars, md *Metadata) (Expression, Type, error) {
	if t, ok := mbType.Unwrap(); ok {
		signature, ok := t.(TypeSignature)
		if !ok {
			return nil, nil, a.NewError(e.cursor, "expected function")
		}
		if len(signature.paramTypes) > 1 {
			signature.returnType = NewSignatureType(signature.cursor, signature.paramTypes[1:], signature.returnType)
			signature.paramTypes = signature.paramTypes[:1]
			return nil, nil, a.NewError(e.cursor, "expected function with one parameter")
		}

		dt, err := signature.paramTypes[0].dereference(typeVars, md)
		if err != nil {
			return nil, nil, err
		}
		record, ok := dt.(typeRecord)
		if !ok {
			return nil, nil, a.NewError(e.cursor, "expected function with one parameter")
		}

		for _, f := range record.fields {
			if f.name == e.name {
				e._type, err = mergeTypes(e.cursor, mbType, a.Just[Type](signature), typeVars, md)
				if err != nil {
					return nil, nil, err
				}

				return e, e._type, nil
			}
		}
	}

	_, err := mergeTypes(e.cursor, mbType, a.Nothing[Type](), typeVars, md)
	return nil, nil, err
}

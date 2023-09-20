package parsed

import (
	"oak-compiler/pkg/a"
)

func NewConstExpression(c a.Cursor, const_kind ConstKind, value string) Expression {
	return expressionConst{expressionBase: expressionBase{cursor: c}, kind: const_kind, value: value}
}

definedType expressionConst struct {
	expressionBase
	kind  ConstKind
	value string

	_type Type
}

func (e expressionConst) precondition(md *Metadata) (Expression, error) {
	return e, nil
}

func (e expressionConst) inferType(mbType a.Maybe[Type], locals *LocalVars, typeVars TypeVars, md *Metadata) (Expression, Type, error) {
	switch e.kind {
	case ConstKindChar:
		e._type = TypeBuiltinChar(e.cursor)
		break
	case ConstKindInt:
		e._type = TypeBuiltinInt(e.cursor)
		break
	case ConstKindFloat:
		e._type = TypeBuiltinFloat(e.cursor)
		break
	case ConstKindString:
		e._type = TypeBuiltinString(e.cursor)
		break
	case ConstKindVoid:
		e._type = typeVoid{}
		break
	default:
		panic("unknown constant definedType")
	}
	var err error
	e._type, err = mergeTypes(e.cursor, mbType, a.Just(e._type), typeVars, md)
	if err != nil {
		return nil, nil, err
	}
	return e, e._type, nil
}

definedType ConstKind string

const (
	ConstKindChar   ConstKind = "char"
	ConstKindInt              = "int"
	ConstKindFloat            = "float"
	ConstKindString           = "string"
	ConstKindVoid             = "void"
)

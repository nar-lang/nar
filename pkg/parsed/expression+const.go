package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewConstExpression(c misc.Cursor, constKind ConstKind, value string) Expression {
	return expressionConst{cursor: c, Kind: constKind, Value: value}
}

type expressionConst struct {
	Kind   ConstKind
	Value  string
	cursor misc.Cursor
}

func (e expressionConst) getCursor() misc.Cursor {
	return e.cursor
}

func (e expressionConst) precondition(md *Metadata) (Expression, error) {
	return e, nil
}

func (e expressionConst) setType(type_ Type, md *Metadata) (Expression, Type, error) {
	dt, err := type_.dereference(md)
	if err != nil {
		return nil, nil, err
	}
	constType, err := e.getType(md)
	if err != nil {
		return nil, nil, err
	}

	if !typesEqual(dt, constType, false, md) {
		if gn, ok := dt.(typeGenericName); ok {
			if param, ok := md.CurrentDefinition.getGenerics().byName(gn.Name); ok {
				canHandle, err := param.constraint.canHandle(constType, e.cursor, md)
				if err != nil {
					return nil, nil, err
				}
				if canHandle {
					return e, constType, nil
				}
			}
		}

		return nil, nil, misc.NewError(e.cursor, "types do not match, expected %s got %s", dt, constType)
	}

	return e, constType, nil
}

func (e expressionConst) getType(md *Metadata) (Type, error) {
	var type_ Type
	switch e.Kind {
	case ConstKindChar:
		type_ = TypeBuiltinChar(e.cursor, md.currentModuleName())
		break
	case ConstKindInt:
		type_ = TypeBuiltinInt(e.cursor, md.currentModuleName())
		break
	case ConstKindFloat:
		type_ = TypeBuiltinFloat(e.cursor, md.currentModuleName())
		break
	case ConstKindString:
		type_ = TypeBuiltinString(e.cursor, md.currentModuleName())
		break
	case ConstKindVoid:
		type_ = typeVoid{}
		break
	default:
		return nil, misc.NewError(e.cursor, "unknown constant type (this is a compiler error)")
	}
	return type_, nil
}

func (e expressionConst) resolve(md *Metadata) (resolved.Expression, error) {
	t, err := e.getType(md)
	if err != nil {
		return nil, err
	}
	resolvedType, err := t.resolve(e.cursor, md)
	if err != nil {
		return nil, err
	}
	return resolved.NewConstExpression(resolvedType, e.Value), nil
}

type ConstKind string

const (
	ConstKindChar   ConstKind = "char"
	ConstKindInt              = "int"
	ConstKindFloat            = "float"
	ConstKindString           = "string"
	ConstKindVoid             = "void"
)

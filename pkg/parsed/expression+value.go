package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

type expressionValue struct {
	ExpressionValue__ int
	Name              string
	InferredType      Type
	cursor            misc.Cursor
}

func (e expressionValue) getCursor() misc.Cursor {
	return e.cursor
}

func (e expressionValue) precondition(md *Metadata) (Expression, error) {
	return e, nil
}

func (e expressionValue) setType(type_ Type, gm genericsMap, md *Metadata) (Expression, Type, error) {
	dt, err := type_.dereference(md)
	if err != nil {
		return nil, nil, err
	}
	valueType, err := md.getTypeByName(md.currentModuleName(), e.Name, nil, e.cursor)
	if err != nil {
		return nil, nil, err
	}

	if !typesEqual(dt, valueType, false, md) {
		if g, ok := dt.(typeGenericNotResolved); ok {
			type_ = valueType
			gm[g.Name] = type_
		} else {
			return nil, nil, misc.NewError(e.cursor, "types do not match, expected %s got %s", dt, valueType)
		}
	}
	e.InferredType = type_
	return e, type_, nil
}

func (e expressionValue) getType(md *Metadata) (Type, error) {
	type_, ok := md.findLocalType(e.Name)
	if !ok {
		return nil, misc.NewError(e.cursor, "unknown identifier")
	}
	return type_, nil
}

func (e expressionValue) resolve(md *Metadata) (resolved.Expression, error) {
	name := e.Name
	type_, ok := md.findLocalType(e.Name)
	if !ok {
		def, ok := md.CurrentModule.definitions[e.Name]
		if !ok {
			return nil, misc.NewError(e.cursor, "unknown value")
		}
		var err error
		type_, err = def.getType(e.cursor, nil, md)
		if err != nil {
			return nil, err
		}
		name = md.CurrentModule.Name() + "_" + name
	}
	resolvedType, err := type_.resolve(e.cursor, md)
	if err != nil {
		return nil, err
	}
	return resolved.NewValueExpression(resolvedType, name), nil
}

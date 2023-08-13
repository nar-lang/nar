package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

type expressionValue struct {
	ExpressionValue__ int
	Name              string
	InferredType      Type
	InferredGenerics  GenericArgs
	cursor            misc.Cursor
}

func (e expressionValue) getCursor() misc.Cursor {
	return e.cursor
}

func (e expressionValue) precondition(md *Metadata) (Expression, error) {
	return e, nil
}

func (e expressionValue) setType(type_ Type, md *Metadata) (Expression, Type, error) {
	dt, err := type_.dereference(md)
	if err != nil {
		return nil, nil, err
	}
	valueType, generics, err := md.getTypeByName(md.currentModuleName(), e.Name, nil, e.cursor)
	if err != nil {
		return nil, nil, err
	}

	gm := valueType.extractGenerics(type_)
	valueType = valueType.mapGenerics(gm)
	generics = generics.mapGenerics(gm)

	if !typesEqual(dt, valueType, false, md) {
		/*if g, ok := dt.(typeGenericNotResolved); ok {
			type_ = valueType
			gm[g.Name] = type_
		} else {
		}*/
		return nil, nil, misc.NewError(e.cursor, "types do not match, expected %s got %s", dt, valueType)
	}
	e.InferredType = type_
	e.InferredGenerics = generics
	return e, type_, nil
}

func (e expressionValue) getType(md *Metadata) (Type, error) {
	type_, _, err := md.getTypeByName(md.currentModuleName(), e.Name, e.InferredGenerics, e.cursor)
	if err != nil {
		return nil, err
	}
	return type_, nil
}

func (e expressionValue) resolve(md *Metadata) (resolved.Expression, error) {
	name := e.Name
	_, local := md.findLocalType(e.Name)
	if e.InferredType == nil {
		return nil, misc.NewError(e.cursor, "trying to resolve not inferred expression value type (this is a compiler error)")
	}

	resolvedType, err := e.InferredType.resolve(e.cursor, md)
	if err != nil {
		return nil, err
	}

	if !local {
		name, err = md.makeRefNameByName(md.currentModuleName(), name, e.cursor)
		if err != nil {
			return nil, err
		}
	}

	resolvedGenerics, err := e.InferredGenerics.resolve(e.cursor, md)
	if err != nil {
		return nil, err
	}
	return resolved.NewValueExpression(resolvedType, name, resolvedGenerics), nil
}

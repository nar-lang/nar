package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

type expressionApply struct {
	Name        string
	Args        Expressions
	ArgTypes    []Type
	RetType     Type
	GenericArgs GenericArgs
	cursor      misc.Cursor
}

func (e expressionApply) getCursor() misc.Cursor {
	return e.cursor
}

func (e expressionApply) precondition(md *Metadata) (Expression, error) {
	var err error
	for i, arg := range e.Args {
		e.Args[i], err = arg.precondition(md)
		if err != nil {
			return nil, err
		}
	}
	return e, nil
}

func (e expressionApply) setType(returnType Type, md *Metadata) (Expression, Type, error) {
	types, inferredReturnType, fnGenerics, err := e.inferTypes(md)
	if err != nil {
		return nil, nil, err
	}

	gm := inferredReturnType.extractGenerics(returnType)
	inferredReturnType = inferredReturnType.mapGenerics(gm)

	for i, arg := range e.Args {
		t, err := arg.getType(md)
		if err != nil {
			return nil, nil, err
		}
		igm := types[i].extractGenerics(t)
		gm = mergeGenericMaps(gm, igm)
	}

	gm = gm.mapSelf()

	for i := range e.Args {
		e.ArgTypes = append(e.ArgTypes, types[i].mapGenerics(gm))
	}

	for i, arg := range e.Args {
		e.Args[i], types[i], err = arg.setType(e.ArgTypes[i], md)
		if err != nil {
			return nil, nil, err
		}
	}
	e.GenericArgs = fnGenerics.mapGenerics(gm)
	e.RetType = inferredReturnType

	return e, e.RetType, nil
}

func (e expressionApply) getType(md *Metadata) (Type, error) {
	if e.RetType == nil {
		_, rt, _, err := e.inferTypes(md)
		if err != nil {
			return nil, err
		}
		return rt, nil
	}
	return e.RetType, nil
}

func (e expressionApply) inferTypes(
	md *Metadata,
) (paramTypes []Type, returnType Type, generics GenericArgs, err error) {
	fnType, fnGenerics, err := md.getTypeByName(md.currentModuleName(), e.Name, nil, e.cursor)
	if err != nil {
		return nil, nil, nil, err
	}
	dt, err := fnType.dereference(md)
	if err != nil {
		return nil, nil, nil, err
	}
	signature, ok := dt.(typeSignature)
	if !ok {
		return nil, nil, nil, misc.NewError(e.cursor, "expected function here")
	}

	paramTypes, returnType = signature.flatten(len(e.Args))
	return paramTypes, returnType, fnGenerics, nil
}

func (e expressionApply) resolve(md *Metadata) (resolved.Expression, error) {
	resolvedArgs, err := e.Args.resolve(md)
	if err != nil {
		return nil, err
	}
	refName, err := md.makeRefNameByName(md.currentModuleName(), e.Name, e.cursor)
	if err != nil {
		return nil, err
	}
	resolvedReturnType, err := e.RetType.resolve(e.cursor, md)
	if err != nil {
		return nil, err
	}
	resolvedGenerics, err := e.GenericArgs.resolve(e.cursor, md)
	if err != nil {
		return nil, err
	}
	var resolvedArgTypes []resolved.Type
	for _, t := range e.ArgTypes {
		rt, err := t.resolve(e.cursor, md)
		if err != nil {
			return nil, err
		}
		resolvedArgTypes = append(resolvedArgTypes, rt)
	}
	return resolved.NewApplyExpression(resolvedReturnType, refName, resolvedGenerics, resolvedArgs, resolvedArgTypes),
		nil
}

package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

type expressionApply struct {
	ExpressionApply__ int
	Name              string
	Args              Expressions
	RetType           Type
	GenericArgs       GenericArgs
	cursor            misc.Cursor
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

func (e expressionApply) setType(returnType Type, gm genericsMap, md *Metadata) (Expression, Type, error) {
	fnType, fnGenerics, err := md.getTypeByName(md.currentModuleName(), e.Name, nil, e.cursor)
	if err != nil {
		return nil, nil, err
	}
	dt, err := fnType.dereference(md)
	if err != nil {
		return nil, nil, err
	}
	signature, ok := dt.(typeSignature)
	if !ok {
		return nil, nil, misc.NewError(e.cursor, "expected function here")
	}

	types, inferredReturnType := signature.flatten(len(e.Args))
	inferredReturnType.extractGenerics(returnType, gm)
	inferredReturnType = inferredReturnType.mapGenerics(gm)
	for i, arg := range e.Args {
		e.Args[i], types[i], err = arg.setType(types[i].mapGenerics(gm), gm, md)
		if err != nil {
			return nil, nil, err
		}
	}
	e.GenericArgs = fnGenerics.mapGenerics(gm)
	e.RetType = inferredReturnType

	return e, e.RetType, nil
}

func (e expressionApply) getType(md *Metadata) (Type, error) {
	return e.RetType, nil
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
	return resolved.NewApplyExpression(resolvedReturnType, refName, resolvedGenerics, resolvedArgs), nil
}

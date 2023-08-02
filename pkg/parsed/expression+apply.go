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

func (e expressionApply) setType(type_ Type, gm genericsMap, md *Metadata) (Expression, Type, error) {
	exprType, err := md.getTypeByName(md.currentModuleName(), e.Name, type_.getGenerics(), e.cursor)
	if err != nil {
		return nil, nil, err
	}
	dt, err := exprType.dereference(md)
	if err != nil {
		return nil, nil, err
	}
	signature, ok := dt.(typeSignature)
	if !ok {
		return nil, nil, misc.NewError(e.cursor, "expected function here")
	}

	types, returnType := signature.flatten(len(e.Args))
	for i, arg := range e.Args {
		e.Args[i], types[i], err = arg.setType(types[i], gm, md)
		if err != nil {
			return nil, nil, err
		}
	}
	e.GenericArgs = exprType.getGenerics()
	e.RetType = returnType.mapGenerics(gm)

	type_.extractGenerics(returnType, gm)
	e.RetType = returnType.mapGenerics(gm)
	inferredType := type_.mapGenerics(gm)
	if !typesEqual(e.RetType, inferredType, false, md) {
		return nil, nil, misc.NewError(e.cursor, "types do not match, expected %s got %s", e.RetType, inferredType)
	}
	e.GenericArgs = e.GenericArgs.mapGenerics(gm)
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

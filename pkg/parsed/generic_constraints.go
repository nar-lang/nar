package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

type GenericConstraint interface {
	resolve(cursor misc.Cursor, md *Metadata) (resolved.GenericConstraint, error)
	canHandle(type_ Type, cursor misc.Cursor, md *Metadata) (bool, error) //TODO: check constraints when instantiating type
}

type GenericConstraintAny struct {
	GenericConstraintAny__ int
}

func (g GenericConstraintAny) canHandle(type_ Type, cursor misc.Cursor, md *Metadata) (bool, error) {
	return true, nil
}

func (g GenericConstraintAny) resolve(cursor misc.Cursor, md *Metadata) (resolved.GenericConstraint, error) {
	return resolved.GenericConstraintAny{}, nil
}

type GenericConstraintType struct {
	GenericConstraintType__ int
	Name                    string
	GenericArgs             GenericArgs
}

func (g GenericConstraintType) resolve(cursor misc.Cursor, md *Metadata) (resolved.GenericConstraint, error) {
	resolvedArgs, err := g.GenericArgs.resolve(cursor, md)
	if err != nil {
		return nil, err
	}
	return resolved.NewTypeGenericConstraint(g.Name, resolvedArgs), nil
}

func (g GenericConstraintType) canHandle(type_ Type, cursor misc.Cursor, md *Metadata) (bool, error) {
	tp, _, err := md.getTypeByName(md.currentModuleName(), g.Name, g.GenericArgs, cursor)
	if err != nil {
		return false, nil
	}
	return typesEqual(type_, tp, false, md), nil
}

type GenericConstraintComparable struct {
	GenericConstraintComparable__ int
}

func (g GenericConstraintComparable) canHandle(type_ Type, cursor misc.Cursor, md *Metadata) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (g GenericConstraintComparable) resolve(cursor misc.Cursor, md *Metadata) (resolved.GenericConstraint, error) {
	return resolved.NewComparableGenericConstraint("runtime.Comparable"), nil
}

type GenericConstraintEquatable struct {
	GenericConstraintEquatable__ int
}

func (g GenericConstraintEquatable) canHandle(type_ Type, cursor misc.Cursor, md *Metadata) (bool, error) {
	dt, err := type_.dereference(md)
	if err != nil {
		return false, err
	}
	_, isSignature := dt.(typeSignature)
	return !isSignature, nil
}

func (g GenericConstraintEquatable) resolve(cursor misc.Cursor, md *Metadata) (resolved.GenericConstraint, error) {
	return resolved.NewEquatableGenericConstraint("runtime.Equatable"), nil
}

type GenericConstraintNumber struct {
	GenericConstraintEquatable__ int
}

func (g GenericConstraintNumber) canHandle(type_ Type, cursor misc.Cursor, md *Metadata) (bool, error) {
	return typesEqual(type_, TypeBuiltinInt(cursor, md.currentModuleName()), false, md) ||
			typesEqual(type_, TypeBuiltinFloat(cursor, md.currentModuleName()), false, md),
		nil
}

func (g GenericConstraintNumber) resolve(cursor misc.Cursor, md *Metadata) (resolved.GenericConstraint, error) {
	return resolved.NewNumberGenericConstraint("runtime.Number"), nil
}

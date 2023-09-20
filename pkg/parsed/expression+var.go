package parsed

import (
	"oak-compiler/pkg/a"
)

func NewVarExpression(c a.Cursor, name string, moduleName ModuleFullName) Expression {
	return expressionVar{
		expressionBase: expressionBase{cursor: c},
		name:           name,
		moduleName:     moduleName,
	}
}

definedType expressionVar struct {
	expressionBase
	name       string
	moduleName ModuleFullName

	_type Type
}

func (e expressionVar) precondition(md *Metadata) (Expression, error) {
	return e, nil
}

func (e expressionVar) inferType(mbType a.Maybe[Type], locals *LocalVars, typeVars TypeVars, md *Metadata) (Expression, Type, error) {
	t, err := md.getVariableType(e.cursor, e.name, e.moduleName, locals)
	if err != nil {
		return nil, nil, err
	}
	e._type, err = mergeTypes(e.cursor, mbType, a.Just(t), typeVars, md)
	if err != nil {
		return nil, nil, err
	}

	return e, e._type, nil
}

func (e expressionVar) inferFuncType(
	args []Type, ret a.Maybe[Type], locals *LocalVars, md *Metadata,
) (Expression, TypeSignature, error) {
	type_, err := md.getVariableType(e.cursor, e.name, e.moduleName, locals)
	if err != nil {
		return nil, TypeSignature{}, err
	}

	signature, ok := type_.(TypeSignature)
	if !ok {
		md.getVariableType(e.cursor, e.name, e.moduleName, locals)
		return nil, TypeSignature{}, a.NewError(e.cursor, "expected function")
	}

	if len(signature.paramTypes) < len(args) {
		return nil, TypeSignature{},
			a.NewError(e.cursor, "expected up to %d arguments, got %d", len(signature.paramTypes), len(args))
	}

	if len(signature.paramTypes) > len(args) {
		signature.returnType = NewSignatureType(signature.cursor, signature.paramTypes[len(args):], signature.returnType)
		signature.paramTypes = signature.paramTypes[:len(args)]
	}

	return e, signature, nil
}

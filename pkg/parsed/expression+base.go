package parsed

import "oak-compiler/pkg/a"

definedType expressionBase struct {
	cursor a.Cursor
}

func (e expressionBase) getCursor() a.Cursor {
	return e.cursor
}

func (e expressionBase) inferFuncType(args []Type, ret a.Maybe[Type], locals *LocalVars, md *Metadata) (Expression, TypeSignature, error) {
	return nil, TypeSignature{}, a.NewError(e.cursor, "expression cannot infer its definedType as a function")
}

definedType expressionDegenerated struct {
	cursor a.Cursor
}

func (e expressionDegenerated) inferType(mbType a.Maybe[Type], locals *LocalVars, typeVars TypeVars, md *Metadata) (Expression, Type, error) {
	panic("expressionDegenerated.inferType() is not allowed")
}

func (e expressionDegenerated) inferFuncType(args []Type, ret a.Maybe[Type], locals *LocalVars, md *Metadata) (Expression, TypeSignature, error) {
	panic("expressionDegenerated.inferFuncType() is not allowed")
}

func (e expressionDegenerated) getCursor() a.Cursor {
	return e.cursor
}

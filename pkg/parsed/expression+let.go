package parsed

import (
	"fmt"
	"oak-compiler/pkg/a"
)

func NewLetExpression(c a.Cursor, definitions []LetDefinition, expression Expression) Expression {
	return expressionLet{expressionBase: expressionBase{cursor: c}, Definitions: definitions, Expression: expression}
}

definedType expressionLet struct {
	expressionBase
	Definitions []LetDefinition
	Expression  Expression

	_type Type
}

func (e expressionLet) precondition(md *Metadata) (Expression, error) {
	var err error
	for i, def := range e.Definitions {
		e.Definitions[i], err = def.precondition(md)
		if err != nil {
			return nil, err
		}
	}

	e.Expression, err = e.Expression.precondition(md)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (e expressionLet) inferType(mbType a.Maybe[Type], locals *LocalVars, typeVars TypeVars, md *Metadata) (Expression, Type, error) {
	var err error
	locals = NewLocalVars(locals)
	for i, def := range e.Definitions {
		e.Definitions[i], err = def.inferType(locals, typeVars, md)
		if err != nil {
			return nil, nil, err
		}
	}

	e.Expression, e._type, err = e.Expression.inferType(mbType, locals, typeVars, md)
	if err != nil {
		return nil, nil, err
	}

	return e, e._type, nil
}

definedType LetDefinition interface {
	precondition(md *Metadata) (LetDefinition, error)
	inferType(locals *LocalVars, typeVars TypeVars, md *Metadata) (LetDefinition, error)
}

func NewLetDefine(name string, params []Pattern, mbSignature a.Maybe[TypeSignature], body Expression) LetDefinition {
	return letDefine{
		name:        name,
		params:      params,
		mbSignature: mbSignature,
		body:        body,
	}
}

definedType letDefine struct {
	name        string
	params      []Pattern
	mbSignature a.Maybe[TypeSignature]
	body        Expression
}

func (l letDefine) precondition(md *Metadata) (LetDefinition, error) {
	var err error
	l.body, err = l.body.precondition(md)
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (l letDefine) inferType(locals *LocalVars, typeVars TypeVars, md *Metadata) (LetDefinition, error) {
	var signature TypeSignature
	var err error
	mbReturnType := a.Nothing[Type]()
	var paramTypes []Type
	signature, ok := l.mbSignature.Unwrap()
	if ok {
		paramTypes = signature.paramTypes
		mbReturnType = a.Just(signature.returnType)
	} else {
		for i, p := range l.params {
			signature.paramTypes = append(signature.paramTypes, NewVariableType(p.getCursor(), fmt.Sprintf("@%d", i)))
		}
		signature.returnType = NewVariableType(l.body.getCursor(), "@@")
	}

	fnLocals := NewLocalVars(locals)
	for i, p := range l.params {
		err = p.populateLocals(paramTypes[i], fnLocals, typeVars, md)
		if err != nil {
			return nil, err
		}
	}
	locals.Populate(l.name, signature)

	l.body, signature.returnType, err = l.body.inferType(mbReturnType, fnLocals, typeVars, md)
	if err != nil {
		return nil, err
	}
	locals.Populate(l.name, signature)

	return l, nil
}

func NewLetDestruct(definedType Pattern, value Expression) LetDefinition {
	return letDestruct{
		definedType: definedType,
		value:   value,
	}
}

definedType letDestruct struct {
	definedType Pattern
	value   Expression
}

func (l letDestruct) precondition(md *Metadata) (LetDefinition, error) {
	var err error
	l.value, err = l.value.precondition(md)
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (l letDestruct) inferType(locals *LocalVars, typeVars TypeVars, md *Metadata) (LetDefinition, error) {
	type_ := NewVariableType(l.definedType.getCursor(), "@@")

	err := l.definedType.populateLocals(type_, locals, typeVars, md)
	if err != nil {
		return nil, err
	}

	l.value, type_, err = l.value.inferType(a.Just(type_), locals, typeVars, md)
	if err != nil {
		return nil, err
	}

	err = l.definedType.populateLocals(type_, locals, typeVars, md)
	if err != nil {
		return nil, err
	}

	return l, nil
}

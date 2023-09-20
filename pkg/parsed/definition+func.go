package parsed

import (
	"fmt"
	"oak-compiler/pkg/a"
)

func NewFuncDefinition(
	c a.Cursor,
	name string,
	hidden, extern bool,
	mbSignature a.Maybe[TypeSignature],
	params []Pattern,
	expression Expression,
) Definition {
	return &definitionFunc{
		definitionBase: definitionBase{
			cursor: c,
			name:   name,
			hidden: hidden,
		},
		expression:  expression,
		mbSignature: mbSignature,
		params:      params,
		extern:      extern,
	}
}

definedType definitionFunc struct {
	definitionBase
	expression  Expression
	mbSignature a.Maybe[TypeSignature]
	params      []Pattern
	extern      bool //TODO: make call expression
}

func (def *definitionFunc) precondition(md *Metadata) error {
	if def.extern {
		return nil
	}

	var err error
	def.expression, err = def.expression.precondition(md)
	if err != nil {
		return err
	}

	return nil
}

func (def *definitionFunc) inferType(md *Metadata) (Type, error) {
	if def._type != nil {
		return def._type, nil
	}

	var err error
	mbReturnType := a.Nothing[Type]()
	var paramTypes []Type
	signature, ok := def.mbSignature.Unwrap()
	if ok {
		paramTypes = signature.paramTypes
		mbReturnType = a.Just(signature.returnType)
	} else {
		for i, p := range def.params {
			signature.paramTypes = append(signature.paramTypes, NewVariableType(p.getCursor(), fmt.Sprintf("@%d", i)))
		}
		signature.returnType = NewVariableType(def.cursor, "@@")
	}

	def._type = signature //TODO: break recursive calls

	locals := NewLocalVars(nil)
	for i, p := range def.params {
		err = p.populateLocals(paramTypes[i], locals, TypeVars{}, md)
		if err != nil {
			return nil, err
		}
	}

	if def.extern {
		if s, ok := def.mbSignature.Unwrap(); ok {
			def._type = s
			return def._type, nil
		}
		return nil, a.NewError(def.cursor, "extern function requires definedType annotation")
	}
	typeVars := TypeVars{}
	def.expression, signature.returnType, err = def.expression.inferType(mbReturnType, locals, typeVars, md)
	//todo: merge definedType vars and replace in expression
	if err != nil {
		return nil, err
	}
	def._type = signature
	return def._type, nil
}

func (def *definitionFunc) getTypeWithParameters(typeParameters []Type, md *Metadata) (Type, error) {
	panic("??")
}

func (def *definitionFunc) nestedDefinitionNames() []string {
	return nil
}

func (def *definitionFunc) unpackNestedDefinitions() []Definition {
	return nil
}

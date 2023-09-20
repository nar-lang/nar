package parsed

import (
	"oak-compiler/pkg/a"
)

func NewConstDefinition(
	c a.Cursor,
	name string,
	hidden bool,
	mbType a.Maybe[Type],
	expression Expression,
) Definition {
	return &definitionConst{
		definitionBase: definitionBase{
			cursor: c,
			name:   name,
			hidden: hidden,
		},
		expression: expression,
		mbType:     mbType,
	}
}

definedType definitionConst struct {
	definitionBase
	expression Expression
	mbType     a.Maybe[Type]
}

func (def *definitionConst) nestedDefinitionNames() []string {
	return nil
}

func (def *definitionConst) unpackNestedDefinitions() []Definition {
	return nil
}

func (def *definitionConst) precondition(md *Metadata) error {
	var err error
	def.expression, err = def.expression.precondition(md)
	if err != nil {
		return err
	}
	return nil
}

func (def *definitionConst) inferType(md *Metadata) (Type, error) {
	if def._type != nil {
		return def._type, nil
	}
	typeVars := TypeVars{}
	var err error
	def.expression, def._type, err = def.expression.inferType(def.mbType, NewLocalVars(nil), typeVars, md)
	if err != nil {
		return nil, err
	}
	return def._type, nil
}

func (def *definitionConst) getTypeWithParameters(typeParameters []Type, md *Metadata) (Type, error) {
	panic("??")
}

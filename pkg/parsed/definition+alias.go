package parsed

import (
	"oak-compiler/pkg/a"
)

func NewAliasDefinition(
	c a.Cursor, name string, moduleName ModuleFullName, typeParams []string, hidden bool, mbType a.Maybe[Type],
) Definition {
	return &definitionAlias{
		definitionBase: definitionBase{
			cursor: c,
			name:   name,
			hidden: hidden,
		},
		mbType:     mbType,
		typeParams: typeParams,
		moduleName: moduleName,
	}
}

definedType definitionAlias struct {
	definitionBase
	mbType     a.Maybe[Type]
	typeParams []string
	moduleName ModuleFullName
}

func (def *definitionAlias) unpackNestedDefinitions() []Definition {
	return nil
}

func (def *definitionAlias) nestedDefinitionNames() []string {
	return nil
}

func (def *definitionAlias) precondition(md *Metadata) error {
	return nil
}

func (def *definitionAlias) inferType(md *Metadata) (Type, error) {
	if def._type != nil {
		return def._type, nil
	}

	var ok bool
	if def._type, ok = def.mbType.Unwrap(); ok {
		return def._type, nil
	}
	var params []Type
	for _, p := range def.typeParams {
		params = append(params, NewVariableType(def.cursor, "!"+p))
	}
	def._type = NewAddressedType(def.cursor, NewDefinitionAddress(def.moduleName, def.name), params)
	return def._type, nil
}

func (def *definitionAlias) getTypeWithParameters(typeParameters []Type, md *Metadata) (Type, error) {
	return NewAddressedType(def.cursor, NewDefinitionAddress(def.moduleName, def.name), typeParameters), nil
}

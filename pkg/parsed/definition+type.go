package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewTypeDefinition(
	c misc.Cursor, address DefinitionAddress, genericParams GenericParams, hidden, extern bool, type_ Type,
) Definition {
	return definitionType{
		definitionBase: definitionBase{
			Address:       address,
			GenericParams: genericParams,
			Hidden:        hidden,
			Extern:        extern,
			cursor:        c,
		},
		Type: type_,
	}
}

type definitionType struct {
	DefinitionType__ int
	definitionBase
	Type Type
}

func (def definitionType) precondition(*Metadata) (Definition, error) {
	return def, nil
}

func (def definitionType) getType(cursor misc.Cursor, generics GenericArgs, md *Metadata) (Type, error) {

	if def.Extern {
		if len(def.GenericParams) != len(generics) {
			return nil, misc.NewError(
				cursor, "expected %d generic arguments, got %d", len(def.GenericParams), len(generics),
			)
		}
		return NewAddressedType(def.cursor, def.Address.moduleFullName, def.Address, generics, true), nil
	}

	gs, err := def.getGenericsMap(cursor, generics)
	if err != nil {
		return nil, err
	}
	return def.Type.mapGenerics(gs), nil
}

func (def definitionType) nestedDefinitionNames() []string {
	return def.Type.nestedDefinitionNames()
}

func (def definitionType) unpackNestedDefinitions() []Definition {
	return def.Type.unpackNestedDefinitions(def)
}

func (def definitionType) resolveName(misc.Cursor, *Metadata) (string, error) {
	return def.Address.moduleFullName.moduleName + "_" + def.Name(), nil
}

func (def definitionType) resolve(md *Metadata) (resolved.Definition, bool, error) {
	if def.Extern {
		return nil, false, nil
	}

	md.CurrentDefinition = def

	resolvedName, err := def.resolveName(def.cursor, md)
	if err != nil {
		return nil, false, err
	}

	resolvedType, err := def.Type.resolve(def.cursor, md)
	if err != nil {
		return nil, false, err
	}

	resolvedParams, err := def.GenericParams.Resolve(md)
	def.GenericParams.Resolve(md)

	return resolved.NewTypeDefinition(resolvedName, resolvedParams, resolvedType), true, nil
}

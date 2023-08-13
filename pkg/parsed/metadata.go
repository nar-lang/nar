package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewMetadata() Metadata {
	return Metadata{
		Packages: map[PackageFullName]Package{},
	}
}

type Metadata struct {
	Packages            map[PackageFullName]Package
	CurrentPackage      Package
	CurrentModule       Module
	CurrentDefinition   Definition
	ImportModuleAliases map[ModuleFullName]string
	LocalVars           map[string]Type
}

func (md *Metadata) getTypeByName(
	enclosingModuleName ModuleFullName, name string, generics GenericArgs, cursor misc.Cursor,
) (Type, GenericArgs, error) {
	if tp, ok := md.findLocalType(name); ok {
		return tp, nil, nil
	}

	address, err := md.getAddressByName(enclosingModuleName, name, cursor)
	if err != nil {
		return nil, nil, err
	}

	return md.getTypeByAddress(address, generics, cursor)
}

func (md *Metadata) getAddressByName(enclosingModuleName ModuleFullName, name string, cursor misc.Cursor) (DefinitionAddress, error) {
	module, ok := md.Packages[enclosingModuleName.packageName].Modules[enclosingModuleName.moduleName]
	if !ok {
		return DefinitionAddress{}, misc.NewError(cursor, "enclosing module not found (this is a compiler error)")
	}

	var address DefinitionAddress
	if _, ok := module.definitions[name]; ok {
		address = DefinitionAddress{
			moduleFullName: enclosingModuleName,
			definitionName: name,
		}
	} else if _, ok := md.CurrentModule.definitions[name]; ok {
		address = DefinitionAddress{
			moduleFullName: md.currentModuleName(),
			definitionName: name,
		}
	} else if address, ok = module.unpackedImports[name]; !ok {
		return DefinitionAddress{}, misc.NewError(cursor, "unknown identifier")
	}

	return address, nil
}

func (md *Metadata) findLocalType(name string) (Type, bool) {
	if tp, ok := md.LocalVars[name]; ok {
		return tp, true
	}
	return nil, false
}

func (md *Metadata) getTypeByAddress(
	address DefinitionAddress, generics GenericArgs, cursor misc.Cursor,
) (Type, GenericArgs, error) {
	if def, ok := md.findDefinitionByAddress(address); ok {
		return def.getType(cursor, generics, md)
	}

	return nil, nil, misc.NewError(cursor, "unknown identifier, dont you forget to import it?")
}

func (md *Metadata) findDefinitionByAddress(address DefinitionAddress) (Definition, bool) {
	if pkg, ok := md.Packages[address.moduleFullName.packageName]; ok {
		if module, ok := pkg.Modules[address.moduleFullName.moduleName]; ok {
			if def, ok := module.definitions[address.definitionName]; ok {
				return def, true
			}
		}
	}
	return nil, false
}

func (md *Metadata) cloneLocalVars() map[string]Type {
	c := map[string]Type{}
	for n, t := range md.LocalVars {
		c[n] = t
	}
	return c
}

func (md *Metadata) currentModuleName() ModuleFullName {
	return ModuleFullName{
		packageName: md.CurrentPackage.FullName(),
		moduleName:  md.CurrentModule.Name(),
	}
}

func (md *Metadata) makeRefNameByAddress(address DefinitionAddress, cursor misc.Cursor) (string, error) {
	def, ok := md.findDefinitionByAddress(address)
	if !ok {
		return "", misc.NewError(cursor, "cannot resolve identifier address")
	}
	name, err := def.resolveName(cursor, md)
	if err != nil {
		return "", err
	}
	if address.moduleFullName == md.currentModuleName() {
		return name, nil
	}

	if address.moduleFullName.packageName != md.CurrentPackage.FullName() {
		return resolved.PackageFullName(md.Packages[address.moduleFullName.packageName].FullName()).SafeName() +
			"." + name, nil
	} else {
		return name, nil
	}
}

func (md *Metadata) makeRefNameByName(
	enclosingModuleName ModuleFullName, name string, cursor misc.Cursor,
) (string, error) {
	if _, ok := md.LocalVars[name]; ok {
		return name, nil
	}

	address, err := md.getAddressByName(enclosingModuleName, name, cursor)
	if err != nil {
		return "", err
	}
	return md.makeRefNameByAddress(address, cursor)
}

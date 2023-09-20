package parsed

import (
	"oak-compiler/pkg/a"
	"oak-compiler/pkg/ast"
)

func NewMetadata(packages map[ast.PackageFullName]Package) *Metadata {
	return &Metadata{Packages: packages}
}

definedType Metadata struct {
	Packages            map[ast.PackageFullName]Package
	CurrentPackage      ast.PackageFullName
	CurrentModule       ModuleFullName
	CurrentDefinition   string
	ImportModuleAliases map[ModuleFullName]string
}

func (md *Metadata) getAddressByName(
	cursor a.Cursor, enclosingModuleName ModuleFullName, name string,
) (DefinitionAddress, error) {
	module, ok := md.Packages[enclosingModuleName.packageName].Modules[enclosingModuleName.moduleName]
	if !ok {
		return DefinitionAddress{}, a.NewError(cursor, "enclosing module not found (this is a parser error)")
	}

	var address DefinitionAddress
	if _, ok := module.definitions[name]; ok {
		address = DefinitionAddress{
			moduleFullName: enclosingModuleName,
			definitionName: name,
		}
	} else if address, ok = module.unpackedImports[name]; !ok {
		return DefinitionAddress{}, a.NewError(cursor, "unknown identifier")
	}

	return address, nil
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

func (md *Metadata) findDefinitionByName(cursor a.Cursor, enclosingModule ModuleFullName, name string) (Definition, error) {
	address, err := md.getAddressByName(cursor, enclosingModule, name)
	if err != nil {
		return nil, err
	}

	if pkg, ok := md.Packages[address.moduleFullName.packageName]; ok {
		if module, ok := pkg.Modules[address.moduleFullName.moduleName]; ok {
			if def, ok := module.definitions[address.definitionName]; ok {
				return def, nil
			}
		}
	}
	return nil, a.NewError(cursor, "cannot find definition")
}

func (md *Metadata) getVariableType(
	cursor a.Cursor, name string, enclosingModule ModuleFullName, locals *LocalVars,
) (Type, error) {
	var err error
	if t, ok := locals.Lookup(name); ok {
		return t, nil
	}

	def, err := md.findDefinitionByName(cursor, enclosingModule, name)
	if err != nil {
		return nil, err
	}

	return def.inferType(md)
}

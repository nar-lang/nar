package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

const (
	kCoreFullPackageName PackageFullName = "github.com/oaklang/core"
	kBasicsModuleName    string          = "Basics"
	kCharModuleName                      = "Char"
	kStringModuleName                    = "String"
	kListModuleName                      = "List"
	kBoolName                            = "Bool"
	kCharName                            = "Char"
	kIntName                             = "Int"
	kFloatName                           = "Float"
	kStringName                          = "String"
	kListName                            = "List"
)

func TypeBuiltinList(c misc.Cursor, enclosingModule ModuleFullName, itemType Type) Type {
	return NewAddressedType(
		c, enclosingModule,
		NewDefinitionAddress(
			ModuleFullName{packageName: kCoreFullPackageName, moduleName: kListModuleName}, kListName,
		),
		GenericArgs{itemType}, false,
	)
}

func TypeBuiltinBool(c misc.Cursor, enclosingModule ModuleFullName) Type {
	return NewAddressedType(
		c, enclosingModule,
		NewDefinitionAddress(
			ModuleFullName{packageName: kCoreFullPackageName, moduleName: kBasicsModuleName}, kBoolName,
		),
		nil, false,
	)
}

func TypeBuiltinChar(c misc.Cursor, enclosingModule ModuleFullName) Type {
	return NewAddressedType(
		c, enclosingModule,
		NewDefinitionAddress(
			ModuleFullName{packageName: kCoreFullPackageName, moduleName: kCharModuleName}, kCharName,
		),
		nil, true,
	)
}

func TypeBuiltinInt(c misc.Cursor, enclosingModule ModuleFullName) Type {
	return NewAddressedType(
		c, enclosingModule,
		NewDefinitionAddress(
			ModuleFullName{packageName: kCoreFullPackageName, moduleName: kBasicsModuleName}, kIntName,
		),
		nil, true,
	)
}

func TypeBuiltinFloat(c misc.Cursor, enclosingModule ModuleFullName) Type {
	return NewAddressedType(
		c, enclosingModule,
		NewDefinitionAddress(
			ModuleFullName{packageName: kCoreFullPackageName, moduleName: kBasicsModuleName}, kFloatName,
		),
		nil, true,
	)
}

func TypeBuiltinString(c misc.Cursor, enclosingModule ModuleFullName) Type {
	return NewAddressedType(
		c, enclosingModule,
		NewDefinitionAddress(
			ModuleFullName{packageName: kCoreFullPackageName, moduleName: kStringModuleName}, kStringName,
		),
		nil, true,
	)
}

type PackageFullName string

func NewModuleFullName(packageName PackageFullName, moduleName string) ModuleFullName {
	return ModuleFullName{
		packageName: packageName,
		moduleName:  moduleName,
	}
}

type ModuleFullName struct {
	packageName PackageFullName
	moduleName  string
}

func NewDefinitionAddress(moduleName ModuleFullName, definitionName string) DefinitionAddress {
	return DefinitionAddress{
		moduleFullName: moduleName,
		definitionName: definitionName,
	}
}

type DefinitionAddress struct {
	moduleFullName ModuleFullName
	definitionName string
}

type Expressions []Expression

func (es Expressions) resolve(md *Metadata) ([]resolved.Expression, error) {
	var result []resolved.Expression
	for _, e := range es {
		re, err := e.resolve(md)
		if err != nil {
			return nil, err
		}
		result = append(result, re)
	}
	return result, nil
}

func (d DefinitionAddress) equalsTo(other DefinitionAddress) bool {
	return other.definitionName == d.definitionName &&
		other.moduleFullName.moduleName == d.moduleFullName.moduleName &&
		other.moduleFullName.packageName == d.moduleFullName.packageName
}

func typesEqual(a, b Type, ignoreGenerics bool, md *Metadata) bool {
	da, err := a.dereference(md)
	if err != nil {
		return false
	}
	db, err := b.dereference(md)
	if err != nil {
		return false
	}

	return da.equalsTo(db, ignoreGenerics, md)
}

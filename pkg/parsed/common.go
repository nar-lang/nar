package parsed

import (
	"fmt"
	"oak-compiler/pkg/a"
	"oak-compiler/pkg/ast"
)

const (
	kCoreFullPackageName ast.PackageFullName = "github.com/oaklang/core/0.1.0"
	kBasicsModuleName    string              = "Basics"
	kCharModuleName                          = "Char"
	kStringModuleName                        = "String"
	kListModuleName                          = "List"
	kBoolName                                = "Bool"
	kCharName                                = "Char"
	kIntName                                 = "Int"
	kFloatName                               = "Float"
	kStringName                              = "String"
	kListName                                = "List"
)

var typeBuiltinListAddress = NewDefinitionAddress(
	ModuleFullName{packageName: kCoreFullPackageName, moduleName: kListModuleName}, kListName,
)

func TypeBuiltinList(c a.Cursor, itemType Type) Type {
	return NewAddressedType(c, typeBuiltinListAddress, []Type{itemType})
}

func ExtractListTypeAndItemType(c a.Cursor, type_ Type, typeVars TypeVars, md *Metadata) (Type, Type, error) {
	dt, err := type_.dereference(typeVars, md)
	if err != nil {
		return typeAddressed{}, nil, err
	}

	addressed, ok := dt.(typeAddressed)

	if !ok || !addressed.address.equalsTo(typeBuiltinListAddress) || len(addressed.typeParams) != 1 {
		return typeAddressed{}, nil, a.NewError(c, "expected list definedType, got %s", type_)
	}

	return addressed, addressed.typeParams[0], nil
}

func TypeBuiltinBool(c a.Cursor) Type {
	return NewAddressedType(
		c,
		NewDefinitionAddress(
			ModuleFullName{packageName: kCoreFullPackageName, moduleName: kBasicsModuleName}, kBoolName,
		),
		nil,
	)
}

func TypeBuiltinChar(c a.Cursor) Type {
	return NewAddressedType(
		c,
		NewDefinitionAddress(
			ModuleFullName{packageName: kCoreFullPackageName, moduleName: kCharModuleName}, kCharName,
		),
		nil,
	)
}

func TypeBuiltinInt(c a.Cursor) Type {
	return NewAddressedType(
		c,
		NewDefinitionAddress(
			ModuleFullName{packageName: kCoreFullPackageName, moduleName: kBasicsModuleName}, kIntName,
		),
		nil,
	)
}

func TypeBuiltinFloat(c a.Cursor) Type {
	return NewAddressedType(
		c,
		NewDefinitionAddress(
			ModuleFullName{packageName: kCoreFullPackageName, moduleName: kBasicsModuleName}, kFloatName,
		),
		nil,
	)
}

func TypeBuiltinString(c a.Cursor) Type {
	return NewAddressedType(
		c,
		NewDefinitionAddress(
			ModuleFullName{packageName: kCoreFullPackageName, moduleName: kStringModuleName}, kStringName,
		),
		nil,
	)
}

func NewModuleFullName(packageName ast.PackageFullName, moduleName string) ModuleFullName {
	return ModuleFullName{
		packageName: packageName,
		moduleName:  moduleName,
	}
}

definedType ModuleFullName struct {
	packageName ast.PackageFullName
	moduleName  string
}

func NewDefinitionAddress(moduleName ModuleFullName, definitionName string) DefinitionAddress {
	return DefinitionAddress{
		moduleFullName: moduleName,
		definitionName: definitionName,
	}
}

definedType DefinitionAddress struct {
	moduleFullName ModuleFullName
	definitionName string
}

func (d DefinitionAddress) equalsTo(other DefinitionAddress) bool {
	return other.definitionName == d.definitionName &&
		other.moduleFullName.moduleName == d.moduleFullName.moduleName &&
		other.moduleFullName.packageName == d.moduleFullName.packageName
}

func (d DefinitionAddress) String() string {
	return fmt.Sprintf("%s/%s.%s", d.moduleFullName.packageName, d.moduleFullName.moduleName, d.definitionName)
}

func mergeTypesAll(cursor a.Cursor, sa, sb []Type, typeVars TypeVars, md *Metadata) ([]Type, error) {
	if len(sa) != len(sb) {
		return nil, a.NewError(cursor, "expected %d definedType parameters, got %d", len(sa), len(sb))
	}

	var merged []Type

	for i, ta := range sa {
		t, err := mergeTypes(cursor, a.Just(ta), a.Just(sb[i]), typeVars, md)
		if err != nil {
			return nil, err
		}
		merged = append(merged, t)
	}
	return merged, nil
}

definedType TypeVars map[string]Type

func mergeTypes(
	cursor a.Cursor, mbTypeA a.Maybe[Type], mbTypeB a.Maybe[Type], typeVars TypeVars, md *Metadata,
) (Type, error) {
	ta, aOk := mbTypeA.Unwrap()
	tb, bOk := mbTypeB.Unwrap()
	if aOk && bOk {
		da, err := ta.dereference(typeVars, md)
		if err != nil {
			return nil, err
		}
		db, err := tb.dereference(typeVars, md)
		if err != nil {
			return nil, err
		}

		if _, ok := db.(typeVariable); ok {
			db, da = da, db
		}

		return da.mergeWith(cursor, db, typeVars, md)
	}
	if aOk {
		return ta, nil
	}
	if bOk {
		return tb, nil
	}
	return nil, a.NewError(cursor, "cannot infer definedType")
}

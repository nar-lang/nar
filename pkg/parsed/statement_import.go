package parsed

import (
	"encoding/json"
	"golang.org/x/exp/slices"
	"oak-compiler/pkg/misc"
)

func NewImportStatement(
	c misc.Cursor, packageName PackageFullName, module, alias string, exposingAll bool, exposing []string,
) StatementImport {
	return StatementImport{
		cursor:      c,
		packageName: packageName,
		moduleName:  module,
		alias:       alias,
		exposing:    exposing,
		exposingAll: exposingAll,
	}
}

type StatementImport struct {
	packageName PackageFullName
	moduleName  string
	alias       string
	exposing    []string
	exposingAll bool
	cursor      misc.Cursor
}

func (imp StatementImport) inject(imports map[string]DefinitionAddress, md *Metadata) (StatementImport, error) {
	packageName := imp.packageName
	if packageName == "" {
		var availablePackageNames []PackageFullName
		for name, pkg := range md.Packages {
			if _, ok := pkg.Modules[imp.moduleName]; ok {
				_, ok := md.CurrentPackage.Info.Dependencies[string(name)]
				if ok || name == md.CurrentPackage.FullName() {
					availablePackageNames = append(availablePackageNames, name)
					break
				}
			}
		}
		if len(availablePackageNames) > 1 {
			return StatementImport{},
				misc.NewError(
					imp.cursor,
					"several packages has module %s, use `from` to determinate which one should be used",
				)
		}
		if len(availablePackageNames) == 0 {
			return StatementImport{},
				misc.NewError(imp.cursor, "cannot find module, is package added to dependencies on oak.json?")
		}
		packageName = availablePackageNames[0]
	}

	imp.packageName = packageName

	for name, d := range md.Packages[imp.packageName].Modules[imp.moduleName].definitions {
		if !d.isHidden() {
			exposing := imp.exposingAll || slices.Contains(imp.exposing, name)
			addImport(imp.packageName, imp.moduleName, imp.alias, name, exposing, imports)
			for _, nestedName := range d.nestedDefinitionNames() {
				addImport(imp.packageName, imp.moduleName, imp.alias, nestedName, exposing, imports)
			}
		}
	}
	return imp, nil
}

func (imp StatementImport) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PackageName PackageFullName
		ModuleName  string
		Alias       string
		Exposing    []string
		ExposingAll bool
	}{
		PackageName: imp.packageName,
		ModuleName:  imp.moduleName,
		Alias:       imp.alias,
		Exposing:    imp.exposing,
		ExposingAll: imp.exposingAll,
	})
}

func addImport(
	packageName PackageFullName, module, alias, name string, exposing bool, imports map[string]DefinitionAddress,
) {
	var identifier string
	if !exposing {
		identifier = alias + "."
	}
	identifier += name
	imports[identifier] = NewDefinitionAddress(
		ModuleFullName{packageName: packageName, moduleName: module}, name,
	)
}

package parsed

import (
	"golang.org/x/exp/slices"
	"oak-compiler/pkg/a"
	"oak-compiler/pkg/ast"
)

func NewImport(
	c a.Cursor, packageName ast.PackageFullName, module, alias string, exposingAll bool, exposing []string,
) Import {
	return Import{
		cursor:      c,
		packageName: packageName,
		moduleName:  module,
		alias:       alias,
		exposing:    exposing,
		exposingAll: exposingAll,
	}
}

definedType Import struct {
	packageName ast.PackageFullName
	moduleName  string
	alias       string
	exposing    []string
	exposingAll bool
	cursor      a.Cursor
}

func (imp Import) inject(imports map[string]DefinitionAddress, md *Metadata) (Import, error) {
	packageName := imp.packageName
	if packageName == "" {
		currentPackage, ok := md.Packages[md.CurrentPackage]
		if !ok {
			panic("current package is not set")
		}

		var availablePackageNames []ast.PackageFullName

		for name := range currentPackage.Modules {
			if name == imp.moduleName {
				availablePackageNames = append(availablePackageNames, md.CurrentPackage)
			}
		}

		for depName, depVersion := range currentPackage.Info.Dependencies {
			pkgName := ast.MakePackageName(depName, depVersion)
			pkg, ok := md.Packages[pkgName]
			if !ok {
				panic("package not loaded")
			}
			for name := range pkg.Modules {
				if name == imp.moduleName {
					availablePackageNames = append(availablePackageNames, pkgName)
				}
			}
		}
		if len(availablePackageNames) > 1 {
			return Import{},
				a.NewError(
					imp.cursor,
					"several packages has module %s, use `from` to determinate which one should be used",
				)
		}
		if len(availablePackageNames) == 0 {
			return Import{},
				a.NewError(imp.cursor, "cannot find module, is package added to dependencies on oak.json?")
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

func addImport(
	packageName ast.PackageFullName, module, alias, name string, exposing bool, imports map[string]DefinitionAddress,
) {
	var identifier string
	if !exposing {
		identifier = alias + "."
	}
	identifier += name
	imports[identifier] = NewDefinitionAddress(NewModuleFullName(packageName, module), name)
	if exposing {
		addImport(packageName, module, alias, name, false, imports)
	}
}

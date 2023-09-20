package parser

import (
	"oak-compiler/pkg/ast"
	"oak-compiler/pkg/parsed"
	"slices"
)

definedType ParsedPackages map[ast.PackageFullName]parsed.Package

definedType CompiledPackages map[ast.PackageFullName]struct{}

func (parsedPackages ParsedPackages) Compile() (CompiledPackages, error) {
	compiledPackages := CompiledPackages{}
	for name := range parsedPackages {
		err := compilePackage(parsedPackages, compiledPackages, name)
		if err != nil {
			return nil, err
		}
	}
	return compiledPackages, nil
}

func compilePackage(
	parsedPackages ParsedPackages, compiledPackages CompiledPackages, packageName ast.PackageFullName,
) error {
	if _, ok := compiledPackages[packageName]; ok {
		return nil
	}

	parsedPackage := parsedPackages[packageName]

	var deps []ast.PackageFullName
	for depName, depVersion := range parsedPackage.Info.Dependencies {
		deps = append(deps, ast.MakePackageName(depName, depVersion))
	}
	slices.Sort(deps)

	for _, dep := range deps {
		err := compilePackage(parsedPackages, compiledPackages, dep)
		if err != nil {
			return err
		}
	}

	md := parsed.NewMetadata(parsedPackages)
	var err error

	parsedPackage, err = parsedPackage.Unpack(md)
	if err != nil {
		return err
	}

	parsedPackage, err = parsedPackage.Precondition(md)
	if err != nil {
		return err
	}

	parsedPackage, err = parsedPackage.InferTypes(md)
	if err != nil {
		return err
	}

	parsedPackages[packageName] = parsedPackage

	//todo: compile
	compiledPackages[packageName] = struct{}{}
	return nil
}

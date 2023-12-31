package processors

import (
	"fmt"
	"oak-compiler/internal/pkg/ast"
	"oak-compiler/internal/pkg/ast/normalized"
	"oak-compiler/internal/pkg/ast/parsed"
	"oak-compiler/internal/pkg/ast/typed"
	"oak-compiler/internal/pkg/common"
	"slices"
)

func Compile(
	pkgNames []ast.PackageIdentifier,
	loadedPackages map[ast.PackageIdentifier]*ast.LoadedPackage,
	parsedModules map[ast.QualifiedIdentifier]*parsed.Module,
	normalizedModules map[ast.QualifiedIdentifier]*normalized.Module,
	typedModules map[ast.QualifiedIdentifier]*typed.Module,
	log *common.LogWriter,
	fileMapper func(modulePath string) string,
) (affectedModuleNames []ast.QualifiedIdentifier) {
	affectedPackages := map[ast.PackageIdentifier]struct{}{}
	for _, pkgName := range pkgNames {
		pkg := loadedPackages[pkgName]
		affectedPackages[pkg.Package.Name] = struct{}{}
		for _, dep := range pkg.Package.Dependencies {
			for _, p := range loadedPackages {
				if _, ok := p.Urls[dep]; ok {
					affectedPackages[p.Package.Name] = struct{}{}
					break
				}
			}
		}
	}

	affectedModules := map[ast.QualifiedIdentifier]struct{}{}
	for pkgName := range affectedPackages {
		pkg := loadedPackages[pkgName]

		referencedPackages := map[ast.PackageIdentifier]struct{}{}
		referencedPackages[pkg.Package.Name] = struct{}{}
		for _, dep := range pkg.Package.Dependencies {
			for _, p := range loadedPackages {
				if _, ok := p.Urls[dep]; ok {
					referencedPackages[p.Package.Name] = struct{}{}
					break
				}
			}
		}

		for _, modulePath := range pkg.Sources {
			var parsedModule *parsed.Module
			for _, m := range parsedModules { //todo: can use hashmap
				if m.Location.FilePath == modulePath {
					parsedModule = m
				}
			}
			if parsedModule == nil {
				var err error
				if doc := fileMapper(modulePath); doc != "" {
					parsedModule, err = ParseWithContent(modulePath, doc)
				} else {
					parsedModule, err = Parse(modulePath)
				}

				if err != nil {
					log.Err(err)
				} else {
					parsedModule.PackageName = pkg.Package.Name
					parsedModule.ReferencedPackages = referencedPackages
					if existedModule, ok := parsedModules[parsedModule.Name]; ok {
						log.Err(common.Error{
							Location: parsedModule.Location,
							Extra:    []ast.Location{existedModule.Location},
							Message:  fmt.Sprintf("module name collision: `%s`", existedModule.Name),
						})
					}
					parsedModules[parsedModule.Name] = parsedModule
				}
			}
			if parsedModule != nil {
				affectedModules[parsedModule.Name] = struct{}{}
			}
		}
	}

	affectedModuleNames = common.Keys(affectedModules)
	slices.Sort(affectedModuleNames)

	if log.HasErrors() {
		return
	}

	for _, name := range affectedModuleNames {
		m := parsedModules[name]
		if err := PreNormalize(m.Name, parsedModules); err != nil {
			log.Err(err)
		}
	}

	if log.HasErrors() {
		return
	}

	for _, name := range affectedModuleNames {
		m := parsedModules[name]
		if err := Normalize(m.Name, parsedModules, normalizedModules); err != nil {
			log.Err(err)
			continue
		}
		if _, ok := typedModules[m.Name]; !ok {
			if err := Solve(m.Name, normalizedModules, typedModules); err != nil {
				log.Err(err)
				continue
			}
			if err := CheckPatterns(typedModules); err != nil {
				log.Err(err)
				continue
			}
		}
	}

	return
}

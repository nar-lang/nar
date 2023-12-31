package processors

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/ast/parsed"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
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
				if m.Location.FilePath() == modulePath {
					parsedModule = m
				}
			}
			if parsedModule == nil {
				var errors []error
				if doc := fileMapper(modulePath); doc != "" {
					parsedModule, errors = ParseWithContent(modulePath, doc)
				} else {
					parsedModule, errors = Parse(modulePath)
				}

				for _, e := range errors {
					log.Err(e)
				}
				if parsedModule == nil {
					continue
				}

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
			if parsedModule != nil {
				affectedModules[parsedModule.Name] = struct{}{}
			}
		}
	}

	affectedModuleNames = common.Keys(affectedModules)
	slices.Sort(affectedModuleNames)

	for _, name := range affectedModuleNames {
		m := parsedModules[name]
		if err := PreNormalize(m.Name, parsedModules); err != nil {
			for _, e := range err {
				log.Err(e)
			}
		}
	}

	for _, name := range affectedModuleNames {
		m := parsedModules[name]
		if err := Normalize(m.Name, parsedModules, normalizedModules); err != nil {
			for _, e := range err {
				log.Err(e)
			}
		}
		if _, ok := typedModules[m.Name]; !ok {
			if err := Solve(m.Name, normalizedModules, typedModules); err != nil {
				for _, e := range err {
					log.Err(e)
				}
			}
			if err := CheckPatterns(typedModules); err != nil {
				for _, e := range err {
					log.Err(e)
				}
			}
		}
	}

	return
}

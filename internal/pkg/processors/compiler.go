package processors

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/ast/parsed"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
	"slices"
	"strconv"
)

var Version = strconv.Itoa(int(common.CompilerVersion)/100) + "." + strconv.Itoa(int(common.CompilerVersion)%100)

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
			for _, m := range parsedModules {
				if m.Location().FilePath() == modulePath {
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

				parsedModule.SetPackageName(pkg.Package.Name)
				parsedModule.SetReferencedPackages(referencedPackages)
				if existedModule, ok := parsedModules[parsedModule.Name()]; ok {
					log.Err(common.NewErrorOf(parsedModule, "module name collision: `%s`", existedModule.Name()))
				}
				parsedModules[parsedModule.Name()] = parsedModule
			}
			if parsedModule != nil {
				affectedModules[parsedModule.Name()] = struct{}{}
			}
		}
	}

	if log.Err() {
		return nil
	}

	affectedModuleNames = common.Keys(affectedModules)
	slices.Sort(affectedModuleNames)

	for _, name := range affectedModuleNames {
		m := parsedModules[name]
		err := m.Generate(parsedModules)
		log.Err(err...)
	}

	if log.Err() {
		return nil
	}

	for _, name := range affectedModuleNames {
		parsedModule := parsedModules[name]
		if err := parsedModule.Normalize(parsedModules, normalizedModules); len(err) > 0 {
			if log.Err(err...) {
				return
			}
			continue
		}

		normalizedModule := normalizedModules[name]
		if err := normalizedModule.Annotate(normalizedModules, typedModules); len(err) > 0 {
			if log.Err(err...) {
				return
			}
			continue
		}

		typedModule := typedModules[name]
		if err := typedModule.CheckTypes(); len(err) > 0 {
			if log.Err(err...) {
				return
			}
			continue
		}

		if err := typedModule.CheckPatterns(); len(err) > 0 {
			if log.Err(err...) {
				return
			}
			continue
		}
	}

	return
}

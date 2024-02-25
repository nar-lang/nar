package processors

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/ast/parsed"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
	"nar-compiler/pkg/locator"
	"nar-compiler/pkg/logger"
	"slices"
	"strconv"
)

var Version = strconv.Itoa(int(common.CompilerVersion)/100) + "." + strconv.Itoa(int(common.CompilerVersion)%100)

func Compile(
	log *logger.LogWriter,
	packages []locator.Package,
	parsedModules map[ast.QualifiedIdentifier]*parsed.Module,
	normalizedModules map[ast.QualifiedIdentifier]*normalized.Module,
	typedModules map[ast.QualifiedIdentifier]*typed.Module,
) (affectedModuleNames []ast.QualifiedIdentifier) {
	affectedModules := map[ast.QualifiedIdentifier]struct{}{}

	for _, pkg := range packages {
		sourceMap := pkg.Sources()
		keys := common.Keys(sourceMap)
		slices.Sort(keys)

		for _, path := range keys {
			var parsedModule *parsed.Module
			for _, m := range parsedModules {
				if m.Location().FilePath() == path {
					parsedModule = m
				}
			}
			if parsedModule == nil {
				var errors []error
				parsedModule, errors = Parse(path, sourceMap[path])
				for _, e := range errors {
					log.Err(e)
				}
				if parsedModule == nil {
					continue
				}
				parsedModule.SetPackageName(ast.PackageIdentifier(pkg.Info().Name))

				referencedPackages := map[ast.PackageIdentifier]struct{}{}
				for p := range pkg.Info().Dependencies {
					referencedPackages[ast.PackageIdentifier(p)] = struct{}{}
				}
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

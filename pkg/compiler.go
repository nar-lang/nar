package narc

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/ast/parsed"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
	"nar-compiler/internal/pkg/processors"
	"nar-compiler/pkg/bytecode"
	"os"
	"path/filepath"
)

func Compile(
	moduleUrls []string, outPath string, debug bool, upgrade bool, cachePath string, log *common.LogWriter,
) (
	packages map[ast.PackageIdentifier]*ast.LoadedPackage, entry ast.FullIdentifier,
) {
	parsedModules := map[ast.QualifiedIdentifier]*parsed.Module{}
	normalizedModules := map[ast.QualifiedIdentifier]*normalized.Module{}
	typedModules := map[ast.QualifiedIdentifier]*typed.Module{}

	bin := bytecode.Binary{
		Exports:   map[bytecode.FullIdentifier]bytecode.Pointer{},
		FuncsMap:  map[bytecode.FullIdentifier]bytecode.Pointer{},
		StringMap: map[string]bytecode.StringHash{},
		ConstMap:  map[bytecode.PackedConst]bytecode.ConstHash{},
	}

	entry = ""
	loadedPackages := map[ast.PackageIdentifier]*ast.LoadedPackage{}
	var requiredPackages []ast.PackageIdentifier

	for _, url := range moduleUrls {
		progress := func(value float32, message string) {
			log.Trace(message)
		}
		loaded, err := processors.LoadPackage(url, cachePath, ".", progress, upgrade, loadedPackages)
		if err != nil {
			log.Err(err)
			continue
		}
		if entry == "" {
			entry = loaded.Package.Main
		}
		requiredPackages = append(requiredPackages, loaded.Package.Name)
	}

	affectedModuleNames := processors.Compile(
		requiredPackages,
		loadedPackages,
		parsedModules,
		normalizedModules,
		typedModules,
		log,
		func(modulePath string) string { return "" })

	if !log.Err() {
		for _, name := range affectedModuleNames {
			m, ok := typedModules[name]
			if !ok {
				log.Err(common.NewSystemError(fmt.Errorf("module '%s' not found", name)))
				continue
			}
			if err := m.Compose(typedModules, debug, &bin); err != nil {
				log.Err(err)
			}
		}
	}

	if !log.Err() {
		outDir := filepath.Dir(outPath)
		err := os.MkdirAll(outDir, os.ModePerm)
		if err != nil {
			log.Err(common.NewSystemError(err))
		} else {
			file, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY, 0640)
			if err != nil {
				log.Err(common.NewSystemError(err))
			} else {
				err := bin.Build(file, common.CompilerVersion, debug)
				if err != nil {
					log.Err(common.NewSystemError(err))
				}
			}
		}
	}

	log.Trace("compilation finished")
	return loadedPackages, entry
}

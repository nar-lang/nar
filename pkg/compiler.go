package oakc

import (
	"fmt"
	"io"
	"oak-compiler/internal/pkg/ast"
	"oak-compiler/internal/pkg/ast/bytecode"
	"oak-compiler/internal/pkg/ast/normalized"
	"oak-compiler/internal/pkg/ast/parsed"
	"oak-compiler/internal/pkg/ast/typed"
	"oak-compiler/internal/pkg/common"
	"oak-compiler/internal/pkg/processors"
	"os"
	"path/filepath"
	"slices"
)

func Compile(moduleUrls []string, outPath string, debug bool, upgrade bool, cachePath string, log io.Writer) (packages []ast.LoadedPackage, err error) {
	//TODO: move everything to canonical error handling without panics
	defer func() {
		x := recover()
		if x != nil {
			if e, ok := x.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", x)
			}
		}
	}()

	parsedModules := map[ast.QualifiedIdentifier]*parsed.Module{}
	normalizedModules := map[ast.QualifiedIdentifier]*normalized.Module{}
	typedModules := map[ast.QualifiedIdentifier]*typed.Module{}

	bin := bytecode.Binary{
		Exports:   map[ast.ExternalIdentifier]bytecode.Pointer{},
		FuncsMap:  map[ast.ExternalIdentifier]bytecode.Pointer{},
		StringMap: map[string]bytecode.StringHash{},
		ConstMap:  map[bytecode.PackedConst]bytecode.ConstHash{},
	}

	var loadedPackages []ast.LoadedPackage
	for _, url := range moduleUrls {
		loadedPackages = processors.LoadPackage(url, cachePath, log, upgrade, loadedPackages)
	}

	for i := len(loadedPackages) - 1; i >= 0; i-- {
		pkg := loadedPackages[i]
		for _, modulePath := range pkg.Sources {
			parsedModule := processors.Parse(modulePath)
			if existedModule, ok := parsedModules[parsedModule.Name]; ok {
				panic(common.SystemError{
					Message: fmt.Sprintf("module name collision: `%s`", existedModule.Name),
				})
			}
			parsedModules[parsedModule.Name] = &parsedModule
		}
	}

	var names []ast.QualifiedIdentifier
	for name := range parsedModules {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		m := parsedModules[name]
		processors.Normalize(m.Name, parsedModules, normalizedModules)
		processors.Solve(m.Name, normalizedModules, typedModules)
		err := processors.CheckPatterns(typedModules)
		if err != nil {
			return nil, err
		}
		processors.Compose(m.Name, typedModules, debug, &bin)
	}

	outDir := filepath.Dir(outPath)
	err = os.MkdirAll(outDir, os.ModePerm)
	if err != nil {
		panic(common.SystemError{Message: err.Error()})
	}

	file, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		panic(common.SystemError{Message: err.Error()})
	}

	bin.Build(file, debug)

	_, _ = fmt.Fprintf(log, "compiled successfully\n")
	return loadedPackages, nil
}

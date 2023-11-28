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
	"strings"
)

func Compile(
	moduleUrls []string, outPath string, debug bool, upgrade bool, cachePath string, log io.Writer,
) (err error, packages []ast.LoadedPackage) {
	defer func() {
		x := recover()
		if x != nil {
			switch x.(type) {
			case common.Error:
				{
					e := x.(common.Error)
					sb := strings.Builder{}
					if e.Location.FilePath != "" {
						line, col := e.Location.GetLineAndColumn()
						sb.WriteString(fmt.Sprintf("%s:%d:%d %s\n", e.Location.FilePath, line, col, e.Message))
					}

					var uniqueExtra []ast.Location
					for _, e := range e.Extra {
						if !slices.ContainsFunc(uniqueExtra, func(x ast.Location) bool {
							return x.FilePath == e.FilePath && x.Position == e.Position
						}) {
							uniqueExtra = append(uniqueExtra, e)
						}
					}

					for _, extra := range uniqueExtra {
						line, col := extra.GetLineAndColumn()
						sb.WriteString(fmt.Sprintf("  -> %s:%d:%d\n", extra.FilePath, line, col))
					}

					if e.Location.FilePath == "" {
						sb.WriteString(fmt.Sprintf("%s\n", e.Message))
					}
					err = fmt.Errorf(sb.String())
					return
				}
			case common.SystemError:
				{
					e := x.(common.SystemError)
					err = fmt.Errorf(e.Message)
					break
				}
			default:
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

	for _, m := range parsedModules {
		processors.Normalize(m.Name, parsedModules, normalizedModules)
		processors.Solve(m.Name, normalizedModules, typedModules)
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
	return nil, loadedPackages
}

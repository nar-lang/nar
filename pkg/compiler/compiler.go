package compiler

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/ast/parsed"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
	"nar-compiler/internal/pkg/processors"
	"nar-compiler/pkg/bytecode"
	"nar-compiler/pkg/linker"
	"nar-compiler/pkg/locator"
	"nar-compiler/pkg/logger"
)

func Compile(log *logger.LogWriter, lc locator.Locator, link linker.Linker, debug bool) *bytecode.Binary {
	parsedModules := map[ast.QualifiedIdentifier]*parsed.Module{}
	normalizedModules := map[ast.QualifiedIdentifier]*normalized.Module{}
	typedModules := map[ast.QualifiedIdentifier]*typed.Module{}

	bin := bytecode.NewBinary()
	hash := bytecode.NewBinaryHash()

	packages, err := lc.Packages()
	if err != nil {
		log.Err(err)
		return bin
	}

	for _, pkg := range packages {
		bin.Packages[bytecode.QualifiedIdentifier(pkg.Info().Name)] = int32(pkg.Info().Version)
	}

	affectedModuleNames := processors.Compile(
		log,
		packages,
		parsedModules,
		normalizedModules,
		typedModules)

	if !log.Err() {
		for _, name := range affectedModuleNames {
			m, ok := typedModules[name]
			if !ok {
				log.Err(common.NewSystemError(fmt.Errorf("module '%s' not found", name)))
				continue
			}
			if err := m.Compose(typedModules, debug, bin, hash); err != nil {
				log.Err(err)
			}
		}
	}

	if !log.Err() {
		if link != nil {
			err := link.Link(log, bin, lc, debug)
			if err != nil {
				log.Err(err)
			}
		}
	}

	log.Trace("compilation finished")
	return bin
}

func Version() int {
	return common.CompilerVersion
}

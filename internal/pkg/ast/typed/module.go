package typed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/bytecode"
	"nar-compiler/internal/pkg/common"
	"slices"
)

type Module struct {
	name         ast.QualifiedIdentifier
	location     ast.Location
	dependencies map[ast.QualifiedIdentifier][]ast.Identifier
	definitions  []*Definition
}

func (module *Module) Location() ast.Location {
	return module.location
}

func NewModule(
	location ast.Location,
	name ast.QualifiedIdentifier,
	dependencies map[ast.QualifiedIdentifier][]ast.Identifier,
	definitions []*Definition,
) *Module {
	return &Module{
		name:         name,
		location:     location,
		dependencies: dependencies,
		definitions:  definitions,
	}
}

func (module *Module) AddDefinition(def *Definition) {
	module.definitions = append(module.definitions, def)
}

func (module *Module) FindDefinition(name ast.Identifier) (*Definition, bool) {
	for _, def := range module.definitions {
		if def.name == name {
			return def, true
		}
	}
	return nil, false
}

func (module *Module) CheckTypes() (errors []error) {
	for _, def := range module.definitions {
		if !def.typed {
			err := def.solveTypes(nil)
			if err != nil {
				errors = append(errors, err)
			}
		}
	}
	return
}

func (module *Module) CheckPatterns() (errors []error) {
	for _, def := range module.definitions {
		err := def.checkPatterns()
		if err != nil {
			errors = append(errors, err)
			continue
		}
	}
	return
}

func (module *Module) Compose(modules map[ast.QualifiedIdentifier]*Module, debug bool, binary *bytecode.Binary) error {
	binary.HashString("")

	if slices.Contains(binary.CompiledPaths, module.name) {
		return nil
	}

	binary.CompiledPaths = append(binary.CompiledPaths, module.name)

	for depModule := range module.dependencies {
		m, ok := modules[depModule]
		if !ok {
			return common.NewErrorOf(module, "module '%s' not found", depModule)
		}
		if err := m.Compose(modules, debug, binary); err != nil {
			return err
		}
	}

	for _, def := range module.definitions {
		extId := common.MakeFullIdentifier(module.name, def.name)
		binary.FuncsMap[extId] = bytecode.Pointer(len(binary.Funcs))
		binary.Funcs = append(binary.Funcs, bytecode.Func{})
	}

	for _, def := range module.definitions {
		pathId := common.MakeFullIdentifier(module.name, def.name)

		ptr := binary.FuncsMap[pathId]
		if binary.Funcs[ptr].Ops == nil {
			binary.Funcs[ptr] = def.Bytecode(pathId, binary)
			if !def.hidden || debug {
				binary.Exports[pathId] = ptr
			}
		}
	}
	return nil
}

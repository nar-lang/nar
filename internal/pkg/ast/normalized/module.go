package normalized

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
)

type Module struct {
	location     ast.Location
	name         ast.QualifiedIdentifier
	dependencies map[ast.QualifiedIdentifier][]ast.Identifier
	definitions  []Definition
}

func NewModule(location ast.Location, name ast.QualifiedIdentifier, definitions []Definition) *Module {
	return &Module{
		name:         name,
		location:     location,
		dependencies: map[ast.QualifiedIdentifier][]ast.Identifier{},
		definitions:  definitions,
	}
}

func (module *Module) Location() ast.Location {
	return module.location
}

func (module *Module) AddDefinition(definition Definition) {
	module.definitions = append(module.definitions, definition)
}

func (module *Module) Dependencies() []ast.QualifiedIdentifier {
	return common.Keys(module.dependencies)
}

func (module *Module) AddDependencies(modName ast.QualifiedIdentifier, identName ast.Identifier) {
	module.dependencies[modName] = append(module.dependencies[modName], identName)
}

func (module *Module) extractLambda(
	loc ast.Location, parentName ast.Identifier, params []Pattern, body Expression,
	locals map[ast.Identifier]Pattern, name ast.Identifier,
) (def Definition, usedLocals []ast.Identifier, replacement Expression) {
	lastLambdaId++
	lambdaName := ast.Identifier(fmt.Sprintf("_lmbd_%s_%d_%s", parentName, lastLambdaId, name))
	paramNames := extractParamNames(params)
	usedLocals = extractUsedLocals(body, locals, paramNames)
	LastDefinitionId++
	localParams := common.Map(func(x ast.Identifier) Pattern { return NewPNamed(loc, nil, x) }, usedLocals)
	params = append(localParams, params...)
	def = NewDefinition(loc, LastDefinitionId, true, lambdaName, loc, params, body, nil)
	module.definitions = append(module.definitions, def)

	replacement = NewGlobal(loc, module.name, def.name())

	if len(usedLocals) > 0 {
		replacement = NewApply(loc, replacement,
			common.Map(func(x ast.Identifier) Expression {
				local, _ := locals[x]
				return NewLocal(loc, x, local, nil) //TODO: nil???
			}, usedLocals),
		)
	}

	return
}

func (module *Module) Annotate(
	modules map[ast.QualifiedIdentifier]*Module,
	typedModules map[ast.QualifiedIdentifier]*typed.Module,
) (errors []error) {
	if _, ok := typedModules[module.name]; ok {
		return
	}

	for depName := range module.dependencies {
		if depName == module.name {
			continue
		}
		depModule, ok := modules[depName]
		if !ok {
			errors = append(errors, common.NewErrorOf(module, "module dependency `%s` not found", depName))
			return
		}
		if err := depModule.Annotate(modules, typedModules); err != nil {
			errors = append(errors, err...)
			return
		}
	}

	o := typed.NewModule(module.location, module.name, module.dependencies, nil)
	typedModules[module.name] = o

	for i := 0; i < len(module.definitions); i++ {
		def := module.definitions[i]
		typedDef, err := def.annotate(modules, typedModules, module.name, nil)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		o.AddDefinition(typedDef)
	}

	return
}

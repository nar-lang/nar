package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
)

func getAnnotatedGlobal(
	moduleName ast.QualifiedIdentifier,
	definitionName ast.Identifier,
	modules map[ast.QualifiedIdentifier]*Module,
	typedModules map[ast.QualifiedIdentifier]*typed.Module,
	stack []*typed.Definition,
	loc ast.Location,
) (*typed.Definition, error) {
	mod, ok := modules[moduleName]
	if !ok {
		return nil, common.NewErrorAt(loc, "module `%s` not found", moduleName)
	}
	nDef, ok := common.Find(
		func(definition Definition) bool {
			return definition.name() == definitionName
		},
		mod.definitions)
	if !ok {
		return nil, common.NewErrorAt(loc, "definition `%s` not found", definitionName)
	}

	def, ok := common.Find(func(definition *typed.Definition) bool {
		return definition.Id() == nDef.id()
	}, stack)

	if !ok {
		typedModule, ok := typedModules[moduleName]
		if !ok {
			errors := mod.Annotate(modules, typedModules)
			if len(errors) > 0 {
				return nil, errors[0]
			}
			typedModule = typedModules[moduleName]
		}

		var err error
		def, ok = typedModule.FindDefinition(nDef.name())
		if !ok {
			def, err = nDef.annotate(modules, typedModules, moduleName, stack)
			if err != nil {
				return def, err
			}
			typedModule.AddDefinition(def)
		}
	}

	return def, nil
}

func extractUsedLocals(
	expr Expression, definedLocals map[ast.Identifier]Pattern, params map[ast.Identifier]Pattern,
) []ast.Identifier {
	usedLocals := map[ast.Identifier]struct{}{}
	expr.extractUsedLocalsSet(definedLocals, usedLocals)
	var uniqueLocals []ast.Identifier
	for k := range usedLocals {
		if _, ok := params[k]; !ok {
			uniqueLocals = append(uniqueLocals, k)
		}
	}
	return uniqueLocals
}

func extractParamNames(params []Pattern) map[ast.Identifier]Pattern {
	paramNames := map[ast.Identifier]Pattern{}
	for _, p := range params {
		p.extractLocals(paramNames)
	}
	return paramNames
}

func annotateTypeSafe(ctx *typed.SolvingContext, type_ Type, params typeParamsMap, source bool) (typed.Type, error) {
	if type_ == nil {
		return nil, nil
	}
	return type_.annotate(ctx, params, source, nil)
}

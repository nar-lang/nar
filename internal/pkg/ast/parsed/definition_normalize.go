package parsed

import (
	"maps"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
)

func normalizeDefinition(
	modules map[ast.QualifiedIdentifier]*Module, module *Module,
	normalizedModule *normalized.Module,
) func(def *Definition) (o *normalized.Definition, params map[ast.Identifier]normalized.Pattern, err error) {
	return func(def *Definition) (o *normalized.Definition, params map[ast.Identifier]normalized.Pattern, err error) {
		if def == nil {
			return nil, nil, nil
		}
		return def.normalize(modules, module, normalizedModule)
	}
}

func (def *Definition) normalize(
	modules map[ast.QualifiedIdentifier]*Module, module *Module,
	normalizedModule *normalized.Module,
) (o *normalized.Definition, params map[ast.Identifier]normalized.Pattern, err error) {

	lastDefinitionId++
	o = &normalized.Definition{
		Id:       lastDefinitionId,
		Name:     def.Name,
		Location: def.Location,
		Hidden:   def.Hidden,
	}
	params = map[ast.Identifier]normalized.Pattern{}
	o.Params, err = common.MapError(normalizePattern(params, modules, module, normalizedModule), def.Params)
	if err != nil {
		return
	}
	locals := maps.Clone(params)
	o.Expression, err = normalizeExpression(locals, modules, module, normalizedModule)(def.Expression)
	if err != nil {
		return
	}
	o.Type, err = normalizeType(modules, module, nil, nil)(def.Type)
	if err != nil {
		return
	}
	return
}

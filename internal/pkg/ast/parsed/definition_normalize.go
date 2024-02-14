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
) (*normalized.Definition, map[ast.Identifier]normalized.Pattern, error) {
	normalized.LastDefinitionId++

	params := map[ast.Identifier]normalized.Pattern{}

	defParams, err1 := common.MapError(normalizePattern(params, modules, module, normalizedModule), def.params)
	locals := maps.Clone(params)
	body, err2 := normalizeExpression(locals, modules, module, normalizedModule)(def.expression)
	type_, err3 := normalizeType(modules, module, nil, nil)(def.type_)

	nDef := normalized.NewDefinition(
		def.location, normalized.LastDefinitionId, def.hidden, def.name, defParams, body, type_)
	return nDef, params, common.MergeErrors(err1, err2, err3)
}

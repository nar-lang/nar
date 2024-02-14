package parsed

import (
	"maps"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
)

type Definition struct {
	location     ast.Location
	hidden       bool
	name         ast.Identifier
	params       []Pattern
	expression   Expression
	declaredType Type
}

func NewDefinition(
	location ast.Location,
	hidden bool,
	name ast.Identifier,
	params []Pattern,
	expression Expression,
	declaredType Type,
) *Definition {
	return &Definition{
		location:     location,
		hidden:       hidden,
		name:         name,
		params:       params,
		expression:   expression,
		declaredType: declaredType,
	}
}

func (def *Definition) GetLocation() ast.Location {
	return def.location
}

func (def *Definition) _parsed() {}

func (def *Definition) normalize(
	modules map[ast.QualifiedIdentifier]*Module, module *Module,
	normalizedModule *normalized.Module,
) (*normalized.Definition, map[ast.Identifier]normalized.Pattern, error) {
	normalized.LastDefinitionId++

	paramLocals := map[ast.Identifier]normalized.Pattern{}
	var params []normalized.Pattern
	var errors []error
	for _, param := range def.params {
		nParam, err := param.normalize(paramLocals, modules, module, normalizedModule)
		errors = append(errors, err)
		params = append(params, nParam)
	}
	locals := maps.Clone(paramLocals)
	body, err := def.expression.normalize(locals, modules, module, normalizedModule)
	errors = append(errors, err)
	var declaredType normalized.Type
	if def.declaredType != nil {
		declaredType, err = def.declaredType.normalize(modules, module, nil, nil)
		errors = append(errors, err)
	}

	nDef := normalized.NewDefinition(
		def.location, normalized.LastDefinitionId, def.hidden, def.name, params, body, declaredType)
	return nDef, paramLocals, common.MergeErrors(errors...)
}

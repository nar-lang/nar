package parsed

import (
	"maps"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
)

type Definition interface {
	Statement
	normalize(
		modules map[ast.QualifiedIdentifier]*Module, module *Module,
		normalizedModule *normalized.Module,
	) (*normalized.Definition, map[ast.Identifier]normalized.Pattern, error)
	name() ast.Identifier
	hidden() bool
}

func NewDefinition(
	location ast.Location,
	hidden bool,
	name ast.Identifier,
	params []Pattern,
	body Expression,
	declaredType Type,
) Definition {
	return &definition{
		location:     location,
		hidden_:      hidden,
		name_:        name,
		params:       params,
		body:         body,
		declaredType: declaredType,
	}
}

type definition struct {
	location     ast.Location
	hidden_      bool
	name_        ast.Identifier
	params       []Pattern
	body         Expression
	declaredType Type
	successor    *normalized.Definition
}

func (def *definition) hidden() bool {
	return def.hidden_
}

func (def *definition) name() ast.Identifier {
	return def.name_
}

func (def *definition) Successor() normalized.Statement {
	return def.successor
}

func (def *definition) Location() ast.Location {
	return def.location
}

func (def *definition) _parsed() {}

func (def *definition) normalize(
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
	body, err := def.body.normalize(locals, modules, module, normalizedModule)
	errors = append(errors, err)
	var declaredType normalized.Type
	if def.declaredType != nil {
		declaredType, err = def.declaredType.normalize(modules, module, nil)
		errors = append(errors, err)
	}

	nDef := normalized.NewDefinition(
		def.location, normalized.LastDefinitionId, def.hidden_, def.name_, params, body, declaredType)
	def.successor = nDef
	return nDef, paramLocals, common.MergeErrors(errors...)
}

func (def *definition) Iterate(f func(statement Statement)) {
	f(def)
	for _, p := range def.params {
		if p != nil {
			p.Iterate(f)
		}
	}
	if def.declaredType != nil {
		def.declaredType.Iterate(f)
	}
	if def.body != nil {
		def.body.Iterate(f)
	}
}

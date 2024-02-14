package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
)

type Definition struct {
	id           uint64
	name         ast.Identifier
	params       []Pattern
	body         Expression
	declaredType Type
	location     ast.Location
	hidden       bool
}

func NewDefinition(
	location ast.Location, id uint64, hidden bool,
	name ast.Identifier, params []Pattern, body Expression,
	declaredType Type,
) *Definition {
	return &Definition{
		location:     location,
		id:           id,
		name:         name,
		params:       params,
		body:         body,
		declaredType: declaredType,
		hidden:       hidden,
	}
}

func (def *Definition) Location() ast.Location {
	return def.location
}

func (def *Definition) FlattenLambdas(params map[ast.Identifier]Pattern, o *Module) {
	lastLambdaId = 0
	if def.body != nil {
		def.body = def.body.flattenLambdas(def.name, o, params)
	}
}

func (def *Definition) annotate(
	modules map[ast.QualifiedIdentifier]*Module,
	typedModules map[ast.QualifiedIdentifier]*typed.Module,
	moduleName ast.QualifiedIdentifier,
	stack []*typed.Definition,
) (*typed.Definition, error) {
	for _, std := range stack {
		if std.Id() == def.id {
			return std, nil
		}
	}

	typedDef := typed.NewDefinition(def.location, def.id, def.hidden, def.name)
	localTypeParams := typeParamsMap{}

	annotatedDeclaredType, err := annotateTypeSafe(typedDef.SolvingContext(), def.declaredType, typeParamsMap{}, true)
	if err != nil {
		return nil, err
	}
	typedDef.SetDeclaredType(annotatedDeclaredType)

	params, err := common.MapError(
		func(p Pattern) (typed.Pattern, error) {
			return p.annotate(
				typedDef.SolvingContext(), localTypeParams, modules, typedModules, moduleName, true, stack)
		},
		def.params)
	if err != nil {
		return nil, err
	}
	typedDef.SetParams(params)

	stack = append(stack, typedDef)
	if def.body != nil {
		body, err := def.body.annotate(
			typedDef.SolvingContext(), localTypeParams, modules, typedModules, moduleName, stack)
		if err != nil {
			return nil, err
		}
		typedDef.SetExpression(body)
	}
	stack = stack[:len(stack)-1]

	return typedDef, nil
}

package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
)

type Definition interface {
	Statement
	FlattenLambdas(params map[ast.Identifier]Pattern, o *Module)
	name() ast.Identifier
	annotate(
		modules map[ast.QualifiedIdentifier]*Module,
		typedModules map[ast.QualifiedIdentifier]*typed.Module,
		moduleName ast.QualifiedIdentifier,
		stack []*typed.Definition,
	) (*typed.Definition, error)
	params() []Pattern
	body() Expression
	setBody(expr Expression)
	id() uint64
	Params() []Pattern
}

func NewDefinition(
	location ast.Location, id uint64, hidden bool,
	name ast.Identifier, nameLocation ast.Location,
	params []Pattern, body Expression, declaredType Type,
) Definition {
	return &definition{
		location:     location,
		id_:          id,
		name_:        name,
		nameLocation: nameLocation,
		params_:      params,
		body_:        body,
		declaredType: declaredType,
		hidden:       hidden,
	}
}

type definition struct {
	id_          uint64
	name_        ast.Identifier
	params_      []Pattern
	body_        Expression
	declaredType Type
	location     ast.Location
	hidden       bool
	successor    *typed.Definition
	nameLocation ast.Location
}

func (def *definition) Params() []Pattern {
	return def.params_
}

func (def *definition) id() uint64 {
	return def.id_
}

func (def *definition) params() []Pattern {
	return def.params_
}

func (def *definition) body() Expression {
	return def.body_
}

func (def *definition) setBody(expr Expression) {
	def.body_ = expr
}

func (def *definition) name() ast.Identifier {
	return def.name_
}

func (def *definition) Successor() typed.Statement {
	return def.successor
}

func (def *definition) Location() ast.Location {
	return def.location
}

func (def *definition) FlattenLambdas(params map[ast.Identifier]Pattern, o *Module) {
	lastLambdaId = 0
	if def.body_ != nil {
		def.body_ = def.body_.flattenLambdas(def.name_, o, params)
	}
}

func (def *definition) annotate(
	modules map[ast.QualifiedIdentifier]*Module,
	typedModules map[ast.QualifiedIdentifier]*typed.Module,
	moduleName ast.QualifiedIdentifier,
	stack []*typed.Definition,
) (*typed.Definition, error) {
	for _, std := range stack {
		if std.Id() == def.id_ {
			return std, nil
		}
	}

	typedDef := typed.NewDefinition(def.location, def.id_, def.hidden, def.name_, def.nameLocation)
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
		def.params_)
	if err != nil {
		return nil, err
	}
	typedDef.SetParams(params)

	stack = append(stack, typedDef)
	if def.body_ != nil {
		body, err := def.body_.annotate(
			typedDef.SolvingContext(), localTypeParams, modules, typedModules, moduleName, stack)
		if err != nil {
			return nil, err
		}
		typedDef.SetExpression(body)
	}
	stack = stack[:len(stack)-1]

	def.successor = typedDef
	return typedDef, nil
}

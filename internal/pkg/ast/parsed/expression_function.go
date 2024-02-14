package parsed

import (
	"maps"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

type Function struct {
	*expressionBase
	name         ast.Identifier
	nameLocation ast.Location
	params       []Pattern
	body         Expression
	declaredType Type
	nested       Expression
}

func NewFunction(
	location ast.Location, name ast.Identifier, nameLocation ast.Location,
	params []Pattern, body Expression, declaredType Type, nested Expression,
) Expression {
	return &Function{
		expressionBase: newExpressionBase(location),
		name:           name,
		nameLocation:   nameLocation,
		params:         params,
		body:           body,
		declaredType:   declaredType,
		nested:         nested,
	}
}

func (e *Function) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	innerLocals := maps.Clone(locals)
	innerLocals[e.name] = normalized.NewPNamed(e.nameLocation, nil, e.name)
	var params []normalized.Pattern
	for _, param := range e.params {
		nParam, err := param.normalize(innerLocals, modules, module, normalizedModule)
		if err != nil {
			return nil, err
		}
		params = append(params, nParam)
	}
	body, err := e.body.normalize(innerLocals, modules, module, normalizedModule)
	if err != nil {
		return nil, err
	}
	nested, err := e.nested.normalize(innerLocals, modules, module, normalizedModule)
	if err != nil {
		return nil, err
	}
	var declaredType normalized.Type
	if e.declaredType != nil {
		declaredType, err = e.declaredType.normalize(modules, module, nil, nil)
		if err != nil {
			return nil, err
		}
	}
	return e.setSuccessor(normalized.NewFunction(e.location, e.name, params, body, declaredType, nested))
}

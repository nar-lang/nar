package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

func NewLambda(location ast.Location, params []Pattern, returnType Type, body Expression) Expression {
	return &Lambda{
		expressionBase: newExpressionBase(location),
		params:         params,
		return_:        returnType,
		body:           body,
	}
}

type Lambda struct {
	*expressionBase
	params  []Pattern
	return_ Type
	body    Expression
}

func (e *Lambda) SemanticTokens() []ast.SemanticToken {
	return nil
}

func (e *Lambda) Iterate(f func(statement Statement)) {
	f(e)
	for _, param := range e.params {
		if param != nil {
			param.Iterate(f)
		}
	}
	if e.return_ != nil {
		e.return_.Iterate(f)
	}
	if e.body != nil {
		e.body.Iterate(f)
	}
}

func (e *Lambda) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	var params []normalized.Pattern
	for _, param := range e.params {
		nParam, err := param.normalize(locals, modules, module, normalizedModule)
		if err != nil {
			return nil, err
		}
		params = append(params, nParam)
	}
	body, err := e.body.normalize(locals, modules, module, normalizedModule)
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(normalized.NewLambda(e.location, params, body))
}

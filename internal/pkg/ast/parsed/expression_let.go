package parsed

import (
	"maps"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

func NewLet(location ast.Location, pattern Pattern, value, nested Expression) Expression {
	return &Let{
		expressionBase: newExpressionBase(location),
		pattern:        pattern,
		value:          value,
		nested:         nested,
	}
}

type Let struct {
	*expressionBase
	pattern Pattern
	value   Expression
	nested  Expression
}

func (e *Let) SemanticTokens() []ast.SemanticToken {
	return nil
}

func (e *Let) Iterate(f func(statement Statement)) {
	f(e)
	if e.pattern != nil {
		e.pattern.Iterate(f)
	}
	if e.value != nil {
		e.value.Iterate(f)
	}
	if e.nested != nil {
		e.nested.Iterate(f)
	}
}

func (e *Let) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	innerLocals := maps.Clone(locals)
	pattern, err := e.pattern.normalize(innerLocals, modules, module, normalizedModule)
	if err != nil {
		return nil, err
	}
	value, err := e.value.normalize(innerLocals, modules, module, normalizedModule)
	if err != nil {
		return nil, err
	}
	nested, err := e.nested.normalize(innerLocals, modules, module, normalizedModule)
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(normalized.NewLet(e.location, pattern, value, nested))
}

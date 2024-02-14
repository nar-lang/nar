package parsed

import (
	"maps"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

type Let struct {
	*expressionBase
	pattern Pattern
	value   Expression
	nested  Expression
}

func NewLet(location ast.Location, pattern Pattern, value, nested Expression) Expression {
	return &Let{
		expressionBase: newExpressionBase(location),
		pattern:        pattern,
		value:          value,
		nested:         nested,
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

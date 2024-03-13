package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

func NewList(location ast.Location, items []Expression) Expression {
	return &List{
		expressionBase: newExpressionBase(location),
		items:          items,
	}
}

type List struct {
	*expressionBase
	items []Expression
}

func (e *List) SemanticTokens() []ast.SemanticToken {
	return nil
}

func (e *List) Iterate(f func(statement Statement)) {
	f(e)
	for _, item := range e.items {
		if item != nil {
			item.Iterate(f)
		}
	}
}

func (e *List) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	var items []normalized.Expression
	for _, item := range e.items {
		nItem, err := item.normalize(locals, modules, module, normalizedModule)
		if err != nil {
			return nil, err
		}
		items = append(items, nItem)
	}
	return e.setSuccessor(normalized.NewList(e.location, items))
}

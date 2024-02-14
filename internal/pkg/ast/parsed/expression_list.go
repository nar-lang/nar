package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

type List struct {
	*expressionBase
	items []Expression
}

func NewList(location ast.Location, items []Expression) Expression {
	return &List{
		expressionBase: newExpressionBase(location),
		items:          items,
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

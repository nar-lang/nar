package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

func NewConst(location ast.Location, value ast.ConstValue) Expression {
	return &Const{
		expressionBase: newExpressionBase(location),
		value:          value,
	}
}

type Const struct {
	*expressionBase
	value ast.ConstValue
}

func (e *Const) Iterate(f func(statement Statement)) {
	f(e)
}

func (e *Const) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	return e.setSuccessor(normalized.NewConst(e.location, e.value))
}

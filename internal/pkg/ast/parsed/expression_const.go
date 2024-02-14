package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

type Const struct {
	*expressionBase
	value ast.ConstValue
}

func NewConst(location ast.Location, value ast.ConstValue) Expression {
	return &Const{
		expressionBase: newExpressionBase(location),
		value:          value,
	}
}

func (e *Const) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	return e.setSuccessor(normalized.NewConst(e.location, e.value))
}

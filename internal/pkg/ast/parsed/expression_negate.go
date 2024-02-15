package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
)

func NewNegate(location ast.Location, nested Expression) Expression {
	return &Negate{
		expressionBase: newExpressionBase(location),
		nested:         nested,
	}
}

type Negate struct {
	*expressionBase
	nested Expression
}

func (e *Negate) Iterate(f func(statement Statement)) {
	f(e)
	if e.nested != nil {
		e.nested.Iterate(f)
	}
}

func (e *Negate) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	nested, err := e.nested.normalize(locals, modules, module, normalizedModule)
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(normalized.NewApply(
		e.location,
		normalized.NewGlobal(e.location, common.NarBaseMathName, common.NarNegName),
		[]normalized.Expression{nested},
	))
}

package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

func NewAccessor(location ast.Location, fieldName ast.Identifier) Expression {
	return &Accessor{
		expressionBase: newExpressionBase(location),
		fieldName:      fieldName,
	}
}

type Accessor struct {
	*expressionBase
	fieldName ast.Identifier
}

func (e *Accessor) Iterate(f func(statement Statement)) {
	f(e)
}

func (e *Accessor) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	lambda := NewLambda(e.location,
		[]Pattern{NewPNamed(e.location, "x")},
		nil,
		NewAccess(e.location, NewVar(e.location, "x"), e.fieldName))
	nLambda, err := lambda.normalize(locals, modules, module, normalizedModule)
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(nLambda)
}

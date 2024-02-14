package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

type Call struct {
	*expressionBase
	name ast.FullIdentifier
	args []Expression
}

func NewCall(location ast.Location, name ast.FullIdentifier, args []Expression) Expression {
	return &Call{
		expressionBase: newExpressionBase(location),
		name:           name,
		args:           args,
	}
}

func (e *Call) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	var args []normalized.Expression
	for _, arg := range e.args {
		nArg, err := arg.normalize(locals, modules, module, normalizedModule)
		if err != nil {
			return nil, err
		}
		args = append(args, nArg)

	}
	return e.setSuccessor(normalized.NewNativeCall(e.location, e.name, args))
}

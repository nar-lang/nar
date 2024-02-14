package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

type Apply struct {
	*expressionBase
	func_ Expression
	args  []Expression
}

func NewApply(location ast.Location, function Expression, args []Expression) Expression {
	return &Apply{
		expressionBase: newExpressionBase(location),
		func_:          function,
		args:           args,
	}
}

func (e *Apply) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	fn, err := e.func_.normalize(locals, modules, module, normalizedModule)
	if err != nil {
		return nil, err
	}
	var args []normalized.Expression
	for _, arg := range e.args {
		nArg, err := arg.normalize(locals, modules, module, normalizedModule)
		if err != nil {
			return nil, err
		}
		args = append(args, nArg)

	}
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(normalized.NewApply(e.location, fn, args))
}

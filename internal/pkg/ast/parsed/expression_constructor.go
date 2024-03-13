package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

func NewConstructor(
	location ast.Location,
	moduleName ast.QualifiedIdentifier,
	dataName ast.Identifier,
	optionName ast.Identifier,
	nameLocation ast.Location,
	args []Expression,
) Expression {
	return &Constructor{
		expressionBase: newExpressionBase(location),
		moduleName:     moduleName,
		dataName:       dataName,
		optionName:     optionName,
		args:           args,
		nameLocation:   nameLocation,
	}
}

type Constructor struct {
	*expressionBase
	moduleName   ast.QualifiedIdentifier
	dataName     ast.Identifier
	optionName   ast.Identifier
	args         []Expression
	nameLocation ast.Location
}

func (e *Constructor) SemanticTokens() []ast.SemanticToken {
	return []ast.SemanticToken{e.nameLocation.ToToken(ast.TokenTypeEnumMember)}
}

func (e *Constructor) Iterate(f func(statement Statement)) {
	f(e)
	for _, arg := range e.args {
		if arg != nil {
			arg.Iterate(f)
		}
	}
}

func (e *Constructor) normalize(
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

	return e.setSuccessor(normalized.NewConstructor(e.location, e.moduleName, e.dataName, e.optionName, args))
}

package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
)

type Call struct {
	*expressionBase
	name ast.FullIdentifier
	args []Expression
}

func NewNativeCall(loc ast.Location, name ast.FullIdentifier, args []Expression) Expression {
	return &Call{
		expressionBase: newExpressionBase(loc),
		name:           name,
		args:           args,
	}
}

func (e *Call) flattenLambdas(parentName ast.Identifier, m *Module, locals map[ast.Identifier]Pattern) Expression {
	for i, a := range e.args {
		e.args[i] = a.flattenLambdas(parentName, m, locals)
	}
	return e
}

func (e *Call) replaceLocals(replace map[ast.Identifier]Expression) Expression {
	for i, a := range e.args {
		e.args[i] = a.replaceLocals(replace)
	}
	return e
}

func (e *Call) extractUsedLocalsSet(definedLocals map[ast.Identifier]Pattern, usedLocals map[ast.Identifier]struct{}) {
	for _, a := range e.args {
		a.extractUsedLocalsSet(definedLocals, usedLocals)
	}
}

func (e *Call) annotate(ctx *typed.SolvingContext, typeParams typeParamsMap, modules map[ast.QualifiedIdentifier]*Module, typedModules map[ast.QualifiedIdentifier]*typed.Module, moduleName ast.QualifiedIdentifier, stack []*typed.Definition) (typed.Expression, error) {
	args, err := common.MapError(func(e Expression) (typed.Expression, error) {
		return e.annotate(ctx, typeParams, modules, typedModules, moduleName, stack)
	}, e.args)
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(typed.NewCall(ctx, e.location, e.name, args))
}

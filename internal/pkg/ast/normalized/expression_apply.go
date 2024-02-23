package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
)

type Apply struct {
	*expressionBase
	func_ Expression
	args  []Expression
}

func NewApply(loc ast.Location, fn Expression, args []Expression) Expression {
	return &Apply{
		expressionBase: newExpressionBase(loc),
		func_:          fn,
		args:           args,
	}
}

func (e *Apply) flattenLambdas(parentName ast.Identifier, m *Module, locals map[ast.Identifier]Pattern) Expression {
	e.func_ = e.func_.flattenLambdas(parentName, m, locals)
	for i, a := range e.args {
		e.args[i] = a.flattenLambdas(parentName, m, locals)
	}
	return e
}

func (e *Apply) replaceLocals(replace map[ast.Identifier]Expression) Expression {
	e.func_ = e.func_.replaceLocals(replace)
	for i, a := range e.args {
		e.args[i] = a.replaceLocals(replace)
	}
	return e
}

func (e *Apply) extractUsedLocalsSet(definedLocals map[ast.Identifier]Pattern, usedLocals map[ast.Identifier]struct{}) {
	e.func_.extractUsedLocalsSet(definedLocals, usedLocals)
	for _, a := range e.args {
		a.extractUsedLocalsSet(definedLocals, usedLocals)
	}
}

func (e *Apply) annotate(ctx *typed.SolvingContext, typeParams typeParamsMap, modules map[ast.QualifiedIdentifier]*Module, typedModules map[ast.QualifiedIdentifier]*typed.Module, moduleName ast.QualifiedIdentifier, stack []*typed.Definition) (typed.Expression, error) {
	fn, err := e.func_.annotate(ctx, typeParams, modules, typedModules, moduleName, stack)
	if err != nil {
		return nil, err
	}
	args, err := common.MapError(func(e Expression) (typed.Expression, error) {
		return e.annotate(ctx, typeParams, modules, typedModules, moduleName, stack)
	}, e.args)
	if err != nil {
		return nil, err
	}
	apply, err := typed.NewApply(ctx, e.location, fn, args)
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(apply)
}

func (e *Apply) Func() Expression {
	return e.func_
}

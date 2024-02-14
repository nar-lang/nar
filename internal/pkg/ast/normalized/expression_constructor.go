package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
)

type Constructor struct {
	*expressionBase
	moduleName ast.QualifiedIdentifier
	dataName   ast.Identifier
	optionName ast.Identifier
	args       []Expression
}

func NewConstructor(
	loc ast.Location,
	moduleName ast.QualifiedIdentifier,
	dataName ast.Identifier,
	optionName ast.Identifier,
	args []Expression,
) Expression {
	return &Constructor{
		expressionBase: newExpressionBase(loc),
		moduleName:     moduleName,
		dataName:       dataName,
		optionName:     optionName,
		args:           args,
	}
}

func (e *Constructor) flattenLambdas(parentName ast.Identifier, m *Module, locals map[ast.Identifier]Pattern) Expression {
	for i, a := range e.args {
		e.args[i] = a.flattenLambdas(parentName, m, locals)
	}
	return e
}

func (e *Constructor) replaceLocals(replace map[ast.Identifier]Expression) Expression {
	for i, a := range e.args {
		e.args[i] = a.replaceLocals(replace)
	}
	return e
}

func (e *Constructor) extractUsedLocalsSet(definedLocals map[ast.Identifier]Pattern, usedLocals map[ast.Identifier]struct{}) {
	for _, a := range e.args {
		a.extractUsedLocalsSet(definedLocals, usedLocals)
	}
}

func (e *Constructor) annotate(ctx *typed.SolvingContext, typeParams typeParamsMap, modules map[ast.QualifiedIdentifier]*Module, typedModules map[ast.QualifiedIdentifier]*typed.Module, moduleName ast.QualifiedIdentifier, stack []*typed.Definition) (typed.Expression, error) {
	ctorDef, err := getAnnotatedGlobal(e.moduleName, e.optionName, modules, typedModules, stack, e.location)
	if err != nil {
		return nil, err
	}
	t := ctorDef.DeclaredType()
	if len(ctorDef.Params()) > 0 && t != nil {
		t = t.(*typed.TFunc).Return()
	}
	args, err := common.MapError(func(e Expression) (typed.Expression, error) {
		return e.annotate(ctx, typeParams, modules, typedModules, moduleName, stack)
	}, e.args)
	if err != nil {
		return nil, err
	}
	var dt *typed.TData
	if t != nil {
		dt = t.(*typed.TData)
	}
	dataName := common.MakeFullIdentifier(e.moduleName, e.dataName)
	return e.setSuccessor(typed.NewConstructor(ctx, e.location, dataName, e.optionName, dt, args))
}

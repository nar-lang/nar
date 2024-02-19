package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
)

type Global struct {
	*expressionBase
	moduleName     ast.QualifiedIdentifier
	definitionName ast.Identifier
}

func NewGlobal(
	loc ast.Location,
	moduleName ast.QualifiedIdentifier,
	definitionName ast.Identifier,
) Expression {
	return &Global{
		expressionBase: newExpressionBase(loc),
		moduleName:     moduleName,
		definitionName: definitionName,
	}
}

func (e *Global) flattenLambdas(parentName ast.Identifier, m *Module, locals map[ast.Identifier]Pattern) Expression {
	return e
}

func (e *Global) replaceLocals(replace map[ast.Identifier]Expression) Expression {
	return e
}

func (e *Global) extractUsedLocalsSet(definedLocals map[ast.Identifier]Pattern, usedLocals map[ast.Identifier]struct{}) {
}

func (e *Global) annotate(ctx *typed.SolvingContext, typeParams typeParamsMap, modules map[ast.QualifiedIdentifier]*Module, typedModules map[ast.QualifiedIdentifier]*typed.Module, moduleName ast.QualifiedIdentifier, stack []*typed.Definition) (typed.Expression, error) {
	targetDef, err := getAnnotatedGlobal(
		e.moduleName, e.definitionName, modules, typedModules, stack, e.location)
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(typed.NewGlobal(ctx, e.location, e.moduleName, e.definitionName, targetDef))
}

func (e *Global) DefinitionName() (ast.QualifiedIdentifier, ast.Identifier) {
	return e.moduleName, e.definitionName
}

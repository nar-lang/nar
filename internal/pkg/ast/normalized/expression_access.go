package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
)

type Access struct {
	*expressionBase
	record    Expression
	fieldName ast.Identifier
}

func NewAccess(loc ast.Location, record Expression, fieldName ast.Identifier) Expression {
	return &Access{
		expressionBase: newExpressionBase(loc),
		record:         record,
		fieldName:      fieldName,
	}
}

func (e *Access) flattenLambdas(parentName ast.Identifier, m *Module, locals map[ast.Identifier]Pattern) Expression {
	e.record = e.record.flattenLambdas(parentName, m, locals)
	return e
}

func (e *Access) replaceLocals(replace map[ast.Identifier]Expression) Expression {
	e.record = e.record.replaceLocals(replace)
	return e
}

func (e *Access) extractUsedLocalsSet(definedLocals map[ast.Identifier]Pattern, usedLocals map[ast.Identifier]struct{}) {
	e.record.extractUsedLocalsSet(definedLocals, usedLocals)
}

func (e *Access) annotate(ctx *typed.SolvingContext, typeParams typeParamsMap, modules map[ast.QualifiedIdentifier]*Module, typedModules map[ast.QualifiedIdentifier]*typed.Module, moduleName ast.QualifiedIdentifier, stack []*typed.Definition) (typed.Expression, error) {
	record, err := e.record.annotate(ctx, typeParams, modules, typedModules, moduleName, stack)
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(typed.NewAccess(ctx, e.location, e.fieldName, record))
}

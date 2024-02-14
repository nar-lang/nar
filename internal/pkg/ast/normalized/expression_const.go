package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
)

type Const struct {
	*expressionBase
	value ast.ConstValue
}

func NewConst(loc ast.Location, value ast.ConstValue) Expression {
	return &Const{
		expressionBase: newExpressionBase(loc),
		value:          value,
	}
}

func (e *Const) flattenLambdas(parentName ast.Identifier, m *Module, locals map[ast.Identifier]Pattern) Expression {
	return e
}

func (e *Const) replaceLocals(replace map[ast.Identifier]Expression) Expression {
	return e
}

func (e *Const) extractUsedLocalsSet(definedLocals map[ast.Identifier]Pattern, usedLocals map[ast.Identifier]struct{}) {
}

func (e *Const) annotate(ctx *typed.SolvingContext, typeParams typeParamsMap, modules map[ast.QualifiedIdentifier]*Module, typedModules map[ast.QualifiedIdentifier]*typed.Module, moduleName ast.QualifiedIdentifier, stack []*typed.Definition) (typed.Expression, error) {
	return e.setSuccessor(typed.NewConst(ctx, e.location, e.value))
}

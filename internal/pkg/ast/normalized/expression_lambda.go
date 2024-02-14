package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
)

type Lambda struct {
	*expressionBase
	params []Pattern
	body   Expression
}

func NewLambda(loc ast.Location, params []Pattern, body Expression) Expression {
	return &Lambda{
		expressionBase: newExpressionBase(loc),
		params:         params,
		body:           body,
	}
}

func (e *Lambda) flattenLambdas(parentName ast.Identifier, m *Module, locals map[ast.Identifier]Pattern) Expression {
	def, _, replacement := m.extractLambda(e.Location(), parentName, e.params, e.body, locals, "")
	paramNames := extractParamNames(def.params)
	def.body = def.body.flattenLambdas(def.name, m, paramNames)
	return replacement
}

func (e *Lambda) replaceLocals(replace map[ast.Identifier]Expression) Expression {
	e.body = e.body.replaceLocals(replace)
	return e
}

func (e *Lambda) extractUsedLocalsSet(definedLocals map[ast.Identifier]Pattern, usedLocals map[ast.Identifier]struct{}) {
	e.body.extractUsedLocalsSet(definedLocals, usedLocals)
}

func (*Lambda) annotate(ctx *typed.SolvingContext, typeParams typeParamsMap, modules map[ast.QualifiedIdentifier]*Module, typedModules map[ast.QualifiedIdentifier]*typed.Module, moduleName ast.QualifiedIdentifier, stack []*typed.Definition) (typed.Expression, error) {
	return nil, common.NewCompilerError("Lambda should be removed with flattenLambdas() before annotation")
}

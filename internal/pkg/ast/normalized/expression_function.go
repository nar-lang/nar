package normalized

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
)

type Function struct {
	*expressionBase
	name        ast.Identifier
	params      []Pattern
	body        Expression
	fnType      Type
	nested      Expression
	predecessor WithSuccessor
}

func NewFunction(
	loc ast.Location,
	name ast.Identifier,
	params []Pattern,
	body Expression,
	fnType Type,
	nested Expression,
	predecessor WithSuccessor,
) Expression {
	return &Function{
		expressionBase: newExpressionBase(loc),
		name:           name,
		params:         params,
		body:           body,
		fnType:         fnType,
		nested:         nested,
		predecessor:    predecessor,
	}
}

func (e *Function) flattenLambdas(parentName ast.Identifier, m *Module, locals map[ast.Identifier]Pattern) Expression {
	lambdaDef, usedLocals, replacement := m.extractLambda(
		e.location, parentName, e.params, e.body, locals, e.name)

	e.predecessor.SetSuccessor(lambdaDef.body())

	if len(usedLocals) > 0 {
		replName := ast.Identifier(fmt.Sprintf("_lmbd_closrue_%d", lastLambdaId))
		replaceMap := map[ast.Identifier]Expression{}

		var closureArgs []Expression
		for i, arg := range usedLocals {
			closureArgs = append(closureArgs, NewLocal(e.location, arg, lambdaDef.params()[i], e.predecessor))
		}

		const selfName = "_self"
		selfPattern := NewPNamed(e.location, nil, selfName)
		lambdaDef.setBody(NewLet(e.location,
			selfPattern,
			NewApply(e.location, NewGlobal(e.location, m.name, lambdaDef.name()), closureArgs),
			lambdaDef.body()))

		replaceMap[e.name] = NewLocal(e.location, selfName, selfPattern, e.predecessor)
		lambdaDef.setBody(lambdaDef.body().replaceLocals(replaceMap))
		paramNames := extractParamNames(lambdaDef.params())
		lambdaDef.setBody(lambdaDef.body().flattenLambdas(lambdaDef.name(), m, paramNames))

		patternName := NewPNamed(e.location, nil, replName)
		replaceMap[e.name] = NewLocal(e.location, replName, patternName, e.predecessor)
		letNested := e.nested.replaceLocals(replaceMap)
		letNested = letNested.flattenLambdas(parentName, m, locals)
		return NewLet(e.location, patternName, replacement, letNested)
	} else {
		replaceMap := map[ast.Identifier]Expression{}
		replaceMap[e.name] = replacement
		lambdaDef.setBody(lambdaDef.body().replaceLocals(replaceMap))
		paramNames := extractParamNames(lambdaDef.params())
		lambdaDef.setBody(lambdaDef.body().flattenLambdas(lambdaDef.name(), m, paramNames))
		replacedLocals := e.nested.replaceLocals(replaceMap)
		return replacedLocals.flattenLambdas(parentName, m, locals)
	}
}

func (e *Function) replaceLocals(replace map[ast.Identifier]Expression) Expression {
	e.body = e.body.replaceLocals(replace)
	e.nested = e.nested.replaceLocals(replace)
	return e
}

func (e *Function) extractUsedLocalsSet(definedLocals map[ast.Identifier]Pattern, usedLocals map[ast.Identifier]struct{}) {
	e.body.extractUsedLocalsSet(definedLocals, usedLocals)
	e.nested.extractUsedLocalsSet(definedLocals, usedLocals)
}

func (*Function) annotate(ctx *typed.SolvingContext, typeParams typeParamsMap, modules map[ast.QualifiedIdentifier]*Module, typedModules map[ast.QualifiedIdentifier]*typed.Module, moduleName ast.QualifiedIdentifier, stack []*typed.Definition) (typed.Expression, error) {
	return nil, common.NewCompilerError("Function should be removed with flattenLambdas() before annotation")
}

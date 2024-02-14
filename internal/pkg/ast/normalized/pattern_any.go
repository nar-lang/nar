package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
)

type PAny struct {
	*patternBase
}

func NewPAny(loc ast.Location, declaredType Type) Pattern {
	return &PAny{
		patternBase: newPatternBase(loc, declaredType),
	}
}

func (e *PAny) extractLocals(locals map[ast.Identifier]Pattern) {}

func (e *PAny) annotate(ctx *typed.SolvingContext, typeParams typeParamsMap, modules map[ast.QualifiedIdentifier]*Module, typedModules map[ast.QualifiedIdentifier]*typed.Module, moduleName ast.QualifiedIdentifier, typeMapSource bool, stack []*typed.Definition) (typed.Pattern, error) {
	annotatedDeclaredType, err := annotateTypeSafe(ctx, e.declaredType, typeParams, typeMapSource)
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(typed.NewPAny(ctx, e.location, annotatedDeclaredType))
}

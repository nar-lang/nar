package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
)

type PList struct {
	*patternBase
	items []Pattern
}

func NewPList(loc ast.Location, declaredType Type, items []Pattern) Pattern {
	return &PList{
		patternBase: newPatternBase(loc, declaredType),
		items:       items,
	}
}

func (e *PList) extractLocals(locals map[ast.Identifier]Pattern) {
	for _, v := range e.items {
		v.extractLocals(locals)
	}
}

func (e *PList) annotate(ctx *typed.SolvingContext, typeParams typeParamsMap, modules map[ast.QualifiedIdentifier]*Module, typedModules map[ast.QualifiedIdentifier]*typed.Module, moduleName ast.QualifiedIdentifier, typeMapSource bool, stack []*typed.Definition) (typed.Pattern, error) {
	items, err := common.MapError(func(x Pattern) (typed.Pattern, error) {
		return x.annotate(ctx, typeParams, modules, typedModules, moduleName, typeMapSource, stack)
	}, e.items)
	if err != nil {
		return nil, err
	}
	annotatedDeclaredType, err := annotateTypeSafe(ctx, e.declaredType, typeParams, typeMapSource)
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(typed.NewPList(ctx, e.location, annotatedDeclaredType, items))
}
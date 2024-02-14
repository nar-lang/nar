package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
)

type PConst struct {
	*patternBase
	value ast.ConstValue
}

func NewPConst(loc ast.Location, type_ Type, value ast.ConstValue) Pattern {
	return &PConst{
		patternBase: newPatternBase(loc, type_),
		value:       value,
	}
}

func (e *PConst) extractLocals(locals map[ast.Identifier]Pattern) {}

func (e *PConst) annotate(ctx *typed.SolvingContext, typeParams typeParamsMap, modules map[ast.QualifiedIdentifier]*Module, typedModules map[ast.QualifiedIdentifier]*typed.Module, moduleName ast.QualifiedIdentifier, typeMapSource bool, stack []*typed.Definition) (typed.Pattern, error) {
	annotatedDeclaredType, err := annotateTypeSafe(ctx, e.declaredType, typeParams, typeMapSource)
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(typed.NewPConst(ctx, e.location, annotatedDeclaredType, e.value))
}

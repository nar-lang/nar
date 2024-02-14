package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
)

type PNamed struct {
	*patternBase
	name ast.Identifier
}

func NewPNamed(loc ast.Location, declaredType Type, name ast.Identifier) Pattern {
	return &PNamed{
		patternBase: newPatternBase(loc, declaredType),
		name:        name,
	}
}

func (e *PNamed) extractLocals(locals map[ast.Identifier]Pattern) {
	locals[e.name] = e
}

func (e *PNamed) annotate(ctx *typed.SolvingContext, typeParams typeParamsMap, modules map[ast.QualifiedIdentifier]*Module, typedModules map[ast.QualifiedIdentifier]*typed.Module, moduleName ast.QualifiedIdentifier, typeMapSource bool, stack []*typed.Definition) (typed.Pattern, error) {
	annotatedDeclaredType, err := annotateTypeSafe(ctx, e.declaredType, typeParams, typeMapSource)
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(typed.NewPNamed(ctx, e.location, annotatedDeclaredType, e.name))
}

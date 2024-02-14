package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
)

type PAlias struct {
	*patternBase
	alias  ast.Identifier
	nested Pattern
}

func NewPAlias(loc ast.Location, declaredType Type, alias ast.Identifier, nested Pattern) Pattern {
	return &PAlias{
		patternBase: newPatternBase(loc, declaredType),
		alias:       alias,
		nested:      nested,
	}
}

func (e *PAlias) extractLocals(locals map[ast.Identifier]Pattern) {
	locals[e.alias] = e
	e.nested.extractLocals(locals)
}

func (e *PAlias) Alias() ast.Identifier {
	return e.alias
}

func (e *PAlias) Nested() Pattern {
	return e.nested
}

func (e *PAlias) annotate(ctx *typed.SolvingContext, typeParams typeParamsMap, modules map[ast.QualifiedIdentifier]*Module, typedModules map[ast.QualifiedIdentifier]*typed.Module, moduleName ast.QualifiedIdentifier, typeMapSource bool, stack []*typed.Definition) (typed.Pattern, error) {
	nested, err := e.nested.annotate(ctx, typeParams, modules, typedModules, moduleName, typeMapSource, stack)
	if err != nil {
		return nil, err
	}
	annotatedDeclaredType, err := annotateTypeSafe(ctx, e.declaredType, typeParams, typeMapSource)
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(typed.NewPAlias(ctx, e.location, annotatedDeclaredType, e.alias, nested))
}

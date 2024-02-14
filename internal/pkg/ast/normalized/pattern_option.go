package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
)

type POption struct {
	*patternBase
	moduleName     ast.QualifiedIdentifier
	definitionName ast.Identifier
	values         []Pattern
}

func NewPOption(
	loc ast.Location,
	declaredType Type,
	moduleName ast.QualifiedIdentifier,
	definitionName ast.Identifier,
	values []Pattern,
) Pattern {
	return &POption{
		patternBase:    newPatternBase(loc, declaredType),
		moduleName:     moduleName,
		definitionName: definitionName,
		values:         values,
	}
}

func (e *POption) extractLocals(locals map[ast.Identifier]Pattern) {
	for _, v := range e.values {
		v.extractLocals(locals)
	}
}

func (e *POption) annotate(ctx *typed.SolvingContext, typeParams typeParamsMap, modules map[ast.QualifiedIdentifier]*Module, typedModules map[ast.QualifiedIdentifier]*typed.Module, moduleName ast.QualifiedIdentifier, typeMapSource bool, stack []*typed.Definition) (typed.Pattern, error) {
	def, err := getAnnotatedGlobal(
		e.moduleName, e.definitionName, modules, typedModules, stack, e.location)
	if err != nil {
		return nil, err
	}

	args, err := common.MapError(func(x Pattern) (typed.Pattern, error) {
		return x.annotate(ctx, typeParams, modules, typedModules, moduleName, typeMapSource, stack)
	}, e.values)
	if err != nil {
		return nil, err
	}
	annotatedDeclaredType, err := annotateTypeSafe(ctx, e.declaredType, typeParams, typeMapSource)
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(typed.NewPOption(ctx, e.location, annotatedDeclaredType, def, args))
}

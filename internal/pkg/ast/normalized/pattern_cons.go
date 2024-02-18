package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
)

type PCons struct {
	*patternBase
	head, tail Pattern
}

func NewPCons(loc ast.Location, declaredType Type, head, tail Pattern) Pattern {
	return &PCons{
		patternBase: newPatternBase(loc, declaredType),
		head:        head,
		tail:        tail,
	}
}

func (e *PCons) extractLocals(locals map[ast.Identifier]Pattern) {
	e.head.extractLocals(locals)
	e.tail.extractLocals(locals)
}

func (e *PCons) annotate(ctx *typed.SolvingContext, typeParams typeParamsMap, modules map[ast.QualifiedIdentifier]*Module, typedModules map[ast.QualifiedIdentifier]*typed.Module, moduleName ast.QualifiedIdentifier, typeMapSource bool, stack []*typed.Definition) (typed.Pattern, error) {
	head, err := e.head.annotate(ctx, typeParams, modules, typedModules, moduleName, typeMapSource, stack)
	if err != nil {
		return nil, err
	}
	tail, err := e.tail.annotate(ctx, typeParams, modules, typedModules, moduleName, typeMapSource, stack)
	if err != nil {
		return nil, err
	}
	annotatedDeclaredType, err := annotateTypeSafe(ctx, e.declaredType, typeParams, typeMapSource)
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(typed.NewPCons(ctx, e.location, annotatedDeclaredType, head, tail))
}

func (e *PCons) Head() Pattern {
	return e.head
}

func (e *PCons) Tail() Pattern {
	return e.tail
}

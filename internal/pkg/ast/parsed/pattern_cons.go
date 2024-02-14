package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
)

type PCons struct {
	*patternBase
	head, tail Pattern
}

func NewPCons(loc ast.Location, head, tail Pattern) Pattern {
	return &PCons{
		patternBase: newPatternBase(loc),
		head:        head,
		tail:        tail,
	}
}

func (e *PCons) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Pattern, error) {
	head, err1 := e.head.normalize(locals, modules, module, normalizedModule)
	tail, err2 := e.tail.normalize(locals, modules, module, normalizedModule)
	var declaredType normalized.Type
	var err3 error
	if e.declaredType != nil {
		declaredType, err3 = e.declaredType.normalize(modules, module, nil, nil)
	}
	return e.setSuccessor(normalized.NewPCons(e.location, declaredType, head, tail)),
		common.MergeErrors(err1, err2, err3)
}

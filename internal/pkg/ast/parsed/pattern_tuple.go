package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
)

func NewPTuple(loc ast.Location, items []Pattern) Pattern {
	return &PTuple{
		patternBase: newPatternBase(loc),
		items:       items,
	}
}

type PTuple struct {
	*patternBase
	items []Pattern
}

func (e *PTuple) SemanticTokens() []ast.SemanticToken {
	return nil
}

func (e *PTuple) Iterate(f func(statement Statement)) {
	f(e)
	for _, item := range e.items {
		if item != nil {
			item.Iterate(f)
		}
	}
	e.patternBase.Iterate(f)
}

func (e *PTuple) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Pattern, error) {
	var items []normalized.Pattern
	var errors []error
	for _, item := range e.items {
		nItem, err := item.normalize(locals, modules, module, normalizedModule)
		if err != nil {
			errors = append(errors, err)
		}
		items = append(items, nItem)
	}
	var declaredType normalized.Type
	if e.declaredType != nil {
		var err error
		declaredType, err = e.declaredType.normalize(modules, module, nil)
		if err != nil {
			errors = append(errors, err)
		}
	}
	return e.setSuccessor(normalized.NewPTuple(e.location, declaredType, items)),
		common.MergeErrors(errors...)
}

func (t *TTuple) normalize(modules map[ast.QualifiedIdentifier]*Module, module *Module, namedTypes namedTypeMap) (normalized.Type, error) {
	var items []normalized.Type
	for _, item := range t.items {
		nItem, err := item.normalize(modules, module, namedTypes)
		if err != nil {
			return nil, err
		}
		items = append(items, nItem)
	}
	return t.setSuccessor(normalized.NewTTuple(t.location, items))
}

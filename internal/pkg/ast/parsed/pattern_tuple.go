package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
)

type PTuple struct {
	*patternBase
	items []Pattern
}

func NewPTuple(loc ast.Location, items []Pattern) Pattern {
	return &PTuple{
		patternBase: newPatternBase(loc),
		items:       items,
	}
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
		errors = append(errors, err)
		items = append(items, nItem)
	}
	var declaredType normalized.Type
	if e.declaredType != nil {
		var err error
		declaredType, err = e.declaredType.normalize(modules, module, nil, nil)
		errors = append(errors, err)
	}
	return e.setSuccessor(normalized.NewPTuple(e.location, declaredType, items)),
		common.MergeErrors(errors...)
}

func (t *TTuple) normalize(
	modules map[ast.QualifiedIdentifier]*Module, module *Module, typeModule *Module, namedTypes namedTypeMap,
) (normalized.Type, error) {
	var items []normalized.Type
	for _, item := range t.items {
		nItem, err := item.normalize(modules, module, typeModule, namedTypes)
		if err != nil {
			return nil, err
		}
		items = append(items, nItem)
	}
	return t.setSuccessor(normalized.NewTTuple(t.location, items))
}

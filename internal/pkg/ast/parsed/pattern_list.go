package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
)

func NewPList(loc ast.Location, items []Pattern) Pattern {
	return &PList{
		patternBase: newPatternBase(loc),
		items:       items,
	}
}

type PList struct {
	*patternBase
	items []Pattern
}

func (e *PList) Iterate(f func(statement Statement)) {
	f(e)
	for _, item := range e.items {
		if item != nil {
			item.Iterate(f)
		}
	}
	e.patternBase.Iterate(f)
}

func (e *PList) normalize(
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
	return e.setSuccessor(normalized.NewPList(e.location, declaredType, items)),
		common.MergeErrors(errors...)
}

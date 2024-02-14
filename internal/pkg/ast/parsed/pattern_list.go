package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
)

type PList struct {
	*patternBase
	items []Pattern
}

func NewPList(loc ast.Location, items []Pattern) Pattern {
	return &PList{
		patternBase: newPatternBase(loc),
		items:       items,
	}
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

		errors = append(errors, err)
		items = append(items, nItem)
	}
	var declaredType normalized.Type
	if e.declaredType != nil {
		var err error
		declaredType, err = e.declaredType.normalize(modules, module, nil, nil)
		errors = append(errors, err)
	}
	return e.setSuccessor(normalized.NewPList(e.location, declaredType, items)),
		common.MergeErrors(errors...)
}

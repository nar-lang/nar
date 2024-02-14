package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
)

type PAlias struct {
	*patternBase
	alias  ast.Identifier
	nested Pattern
}

func NewPAlias(loc ast.Location, alias ast.Identifier, nested Pattern) Pattern {
	return &PAlias{
		patternBase: newPatternBase(loc),
		alias:       alias,
		nested:      nested,
	}
}

func (e *PAlias) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Pattern, error) {
	nested, err1 := e.nested.normalize(locals, modules, module, normalizedModule)
	var declaredType normalized.Type
	var err2 error
	if e.declaredType != nil {
		declaredType, err2 = e.declaredType.normalize(modules, module, nil, nil)
	}
	np := normalized.NewPAlias(e.location, declaredType, e.alias, nested)
	locals[e.alias] = np
	return e.setSuccessor(np), common.MergeErrors(err1, err2)
}

package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

type PAny struct {
	*patternBase
}

func NewPAny(loc ast.Location) Pattern {
	return &PAny{
		patternBase: newPatternBase(loc),
	}
}

func (e *PAny) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Pattern, error) {
	var declaredType normalized.Type
	var err error
	if e.declaredType != nil {
		declaredType, err = e.declaredType.normalize(modules, module, nil, nil)
	}
	return e.setSuccessor(normalized.NewPAny(e.location, declaredType)), err
}

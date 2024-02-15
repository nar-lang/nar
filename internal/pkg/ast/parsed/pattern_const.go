package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

func NewPConst(loc ast.Location, value ast.ConstValue) Pattern {
	return &PConst{
		patternBase: newPatternBase(loc),
		value:       value,
	}
}

type PConst struct {
	*patternBase
	value ast.ConstValue
}

func (e *PConst) Iterate(f func(statement Statement)) {
	f(e)
	e.patternBase.Iterate(f)
}

func (e *PConst) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Pattern, error) {
	var declaredType normalized.Type
	var err error
	if e.declaredType != nil {
		declaredType, err = e.declaredType.normalize(modules, module, nil)
	}
	return e.setSuccessor(normalized.NewPConst(e.location, declaredType, e.value)), err
}

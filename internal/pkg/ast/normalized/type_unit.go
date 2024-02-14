package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
)

type TUnit struct {
	*typeBase
}

func NewTUnit(loc ast.Location) Type {
	return &TUnit{
		typeBase: newTypeBase(loc),
	}
}

func (e *TUnit) annotate(ctx *typed.SolvingContext, params typeParamsMap, source bool, placeholders placeholderMap) (typed.Type, error) {
	return e.setSuccessor(typed.NewTNative(e.location, common.NarBaseBasicsUnit, nil))
}

package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/bytecode"
	"nar-compiler/internal/pkg/common"
)

type Const struct {
	*expressionBase
	value ast.ConstValue
}

func NewConst(ctx *SolvingContext, loc ast.Location, value ast.ConstValue) Expression {
	return ctx.annotateExpression(&Const{
		expressionBase: &expressionBase{
			location: loc,
		},
		value: value,
	})
}

func (e *Const) checkPatterns() error {
	return nil
}

func (e *Const) mapTypes(subst map[uint64]Type) error {
	var err error
	e.type_, err = e.type_.mapTo(subst)
	return err
}

func (e *Const) Code(currentModule ast.QualifiedIdentifier) string {
	return fmt.Sprintf("%s", e.value.Code(currentModule))
}

func (e *Const) appendEquations(eqs Equations, loc *ast.Location, localDefs localTypesMap, ctx *SolvingContext, stack []*Definition) (Equations, error) {
	return append(eqs, NewEquation(e, e.type_, getConstType(ctx, e.value, e))), nil
}

func (e *Const) appendBytecode(ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary) ([]bytecode.Op, []ast.Location) {
	v := e.value
	if iv, ok := v.(ast.CInt); ok {
		if ex, ok := e.type_.(*TNative); ok && ex.name == common.NarBaseMathFloat {
			v = ast.CFloat{Value: float64(iv.Value)}
		}
	}
	return bytecode.AppendLoadConstValue(v, bytecode.StackKindObject, e.location, ops, locations, binary)
}

package typed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/bytecode"
	"nar-compiler/internal/pkg/common"
)

type Local struct {
	*expressionBase
	name   ast.Identifier
	target Pattern
}

func NewLocal(ctx *SolvingContext, loc ast.Location, name ast.Identifier, target Pattern) Expression {
	return ctx.annotateExpression(&Local{
		expressionBase: newExpressionBase(loc),
		name:           name,
		target:         target,
	})
}

func (e *Local) checkPatterns() error {
	return nil
}

func (e *Local) mapTypes(subst map[uint64]Type) error {
	var err error
	e.type_, err = e.type_.mapTo(subst)
	if err != nil {
		return err
	}
	return nil
}

func (e *Local) Code(currentModule ast.QualifiedIdentifier) string {
	return string(e.name)
}

func (e *Local) appendEquations(eqs Equations, loc *ast.Location, localDefs localTypesMap, ctx *SolvingContext, stack []*Definition) (Equations, error) {
	if e.target != nil {
		eqs = append(eqs, NewEquation(e, e.type_, e.target.Type()))
	} else {
		return nil, common.NewErrorOf(e, "local `%s` not found", e.name)
	}
	return eqs, nil
}

func (e *Local) appendBytecode(ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary) ([]bytecode.Op, []ast.Location) {
	return bytecode.AppendLoadLocal(string(e.name), e.location, ops, locations, binary)
}

func (e *Local) Target() Pattern {
	return e.target
}

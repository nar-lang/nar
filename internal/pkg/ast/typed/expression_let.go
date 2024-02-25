package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	bytecode "nar-compiler/pkg/bytecode"
)

type Let struct {
	*expressionBase
	pattern Pattern
	value   Expression
	body    Expression
}

func NewLet(ctx *SolvingContext, loc ast.Location, pattern Pattern, value Expression, body Expression) Expression {
	return ctx.annotateExpression(&Let{
		expressionBase: newExpressionBase(loc),
		pattern:        pattern,
		value:          value,
		body:           body,
	})
}

func (e *Let) checkPatterns() error {
	if err := checkPattern(e.pattern); err != nil {
		return err
	}
	if err := e.value.checkPatterns(); err != nil {
		return err
	}
	return e.body.checkPatterns()
}

func (e *Let) mapTypes(subst map[uint64]Type) error {
	var err error
	e.type_, err = e.type_.mapTo(subst)
	if err != nil {
		return err
	}
	err = e.pattern.mapTypes(subst)
	if err != nil {
		return err
	}
	err = e.value.mapTypes(subst)
	if err != nil {
		return err
	}
	return e.body.mapTypes(subst)
}

func (e *Let) Children() []Statement {
	return append(e.expressionBase.Children(), e.pattern, e.value, e.body)
}

func (e *Let) Code(currentModule ast.QualifiedIdentifier) string {
	return fmt.Sprintf("let %s = %s in %s",
		e.pattern.Code(currentModule),
		e.value.Code(currentModule),
		e.body.Code(currentModule))
}

func (e *Let) appendEquations(eqs Equations, loc *ast.Location, localDefs localTypesMap, ctx *SolvingContext, stack []*Definition) (Equations, error) {
	var err error
	eqs = append(eqs, NewEquation(e, e.type_, e.body.Type()))
	eqs, err = e.pattern.appendEquations(eqs, loc, localDefs, ctx, stack)
	if err != nil {
		return nil, err
	}

	eqs, err = e.value.appendEquations(eqs, loc, localDefs, ctx, stack)
	if err != nil {
		return nil, err
	}

	eqs = append(eqs, NewEquation(e, e.pattern.Type(), e.value.Type()))

	eqs, err = e.body.appendEquations(eqs, loc, localDefs, ctx, stack)
	if err != nil {
		return nil, err
	}
	return eqs, nil
}

func (e *Let) appendBytecode(ops []bytecode.Op, locations []bytecode.Location, binary *bytecode.Binary, hash *bytecode.BinaryHash) ([]bytecode.Op, []bytecode.Location) {
	ops, locations = e.value.appendBytecode(ops, locations, binary, hash)
	ops, locations = e.pattern.appendBytecode(ops, locations, binary, hash)
	ops, locations = bytecode.AppendJump(0, true, e.location.Bytecode(), ops, locations)
	ops, locations = bytecode.AppendSwapPop(e.location.Bytecode(), bytecode.SwapPopModePop, ops, locations)
	return e.body.appendBytecode(ops, locations, binary, hash)
}

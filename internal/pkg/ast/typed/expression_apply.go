package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/bytecode"
	"nar-compiler/internal/pkg/common"
)

type Apply struct {
	*expressionBase
	func_ Expression
	args  []Expression
}

func NewApply(ctx *SolvingContext, loc ast.Location, func_ Expression, args []Expression) Expression {
	return ctx.annotateExpression(&Apply{
		expressionBase: newExpressionBase(loc),
		func_:          func_,
		args:           args,
	})
}

func (e *Apply) checkPatterns() error {
	if err := e.func_.checkPatterns(); err != nil {
		return err
	}
	for _, arg := range e.args {
		if err := arg.checkPatterns(); err != nil {
			return err
		}
	}
	return nil
}

func (e *Apply) mapTypes(subst map[uint64]Type) error {
	var err error
	e.type_, err = e.type_.mapTo(subst)
	if err != nil {
		return err
	}
	for _, arg := range e.args {
		err = arg.mapTypes(subst)
		if err != nil {
			return err
		}
	}
	return e.func_.mapTypes(subst)
}

func (e *Apply) Children() []Statement {
	children := e.expressionBase.Children()
	children = append(children, e.func_)
	return append(common.Map(func(x Expression) Statement { return x }, e.args))
}

func (e *Apply) Code(currentModule ast.QualifiedIdentifier) string {
	return fmt.Sprintf("%s(%s)", e.func_.Code(currentModule), common.Fold(
		func(x Expression, s string) string {
			if s != "" {
				s += ", "
			}
			return s + x.Code(currentModule)
		}, "", e.args))
}

func (e *Apply) appendEquations(eqs Equations, loc *ast.Location, localDefs localTypesMap, ctx *SolvingContext, stack []*Definition) (Equations, error) {
	var err error
	funcType := NewTFunc(e.location, common.Map(func(p Expression) Type { return p.Type() }, e.args), e.type_)
	eqs = append(eqs, NewEquation(e, e.func_.Type(), funcType))
	eqs, err = e.func_.appendEquations(eqs, loc, localDefs, ctx, stack)
	if err != nil {
		return nil, err
	}
	for _, arg := range e.args {
		eqs, err = arg.appendEquations(eqs, loc, localDefs, ctx, stack)
		if err != nil {
			return nil, err
		}
	}
	return eqs, nil
}

func (e *Apply) appendBytecode(ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary) ([]bytecode.Op, []ast.Location) {
	var err error
	for _, arg := range e.args {
		ops, locations = arg.appendBytecode(ops, locations, binary)
		if err != nil {
			return nil, nil
		}
	}
	ops, locations = e.func_.appendBytecode(ops, locations, binary)
	if err != nil {
		return nil, nil
	}
	return bytecode.AppendApply(len(e.args), e.location, ops, locations)
}

func (e *Apply) Func() Expression {
	return e.func_
}

package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
	bytecode "nar-compiler/pkg/bytecode"
)

type Call struct {
	*expressionBase
	name ast.FullIdentifier
	args []Expression
}

func NewCall(ctx *SolvingContext, loc ast.Location, name ast.FullIdentifier, args []Expression) (Expression, error) {
	if len(args) > 255 {
		return nil, common.NewErrorAt(loc, "too many arguments (max 255)")
	}
	return ctx.annotateExpression(&Call{
		expressionBase: newExpressionBase(loc),
		name:           name,
		args:           args,
	}), nil
}

func (e *Call) checkPatterns() error {
	for _, arg := range e.args {
		if err := arg.checkPatterns(); err != nil {
			return err
		}
	}
	return nil
}

func (e *Call) mapTypes(subst map[uint64]Type) error {
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
	return nil
}

func (e *Call) Children() []Statement {
	return append(e.expressionBase.Children(), common.Map(func(x Expression) Statement { return x }, e.args)...)
}

func (e *Call) Code(currentModule ast.QualifiedIdentifier) string {
	return fmt.Sprintf("%s(%s)", e.name, common.Fold(
		func(x Expression, s string) string {
			if s != "" {
				s += ", "
			}
			return s + x.Code(currentModule)
		}, "", e.args))
}

func (e *Call) appendEquations(eqs Equations, loc *ast.Location, localDefs localTypesMap, ctx *SolvingContext, stack []*Definition) (Equations, error) {
	var err error
	for _, a := range e.args {
		eqs, err = a.appendEquations(eqs, loc, localDefs, ctx, stack)
		if err != nil {
			return nil, err
		}
	}
	return eqs, nil
}

func (e *Call) appendBytecode(ops []bytecode.Op, locations []bytecode.Location, binary *bytecode.Binary) ([]bytecode.Op, []bytecode.Location) {
	for _, arg := range e.args {
		ops, locations = arg.appendBytecode(ops, locations, binary)
	}
	ops, locations = bytecode.AppendCall(string(e.name), uint8(len(e.args)), e.location.Bytecode(), ops, locations, binary)
	return ops, locations
}

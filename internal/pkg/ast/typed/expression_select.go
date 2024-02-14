package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/bytecode"
	"nar-compiler/internal/pkg/common"
)

type Select struct {
	*expressionBase
	condition Expression
	cases     []*SelectCase
}

func NewSelect(ctx *SolvingContext, loc ast.Location, condition Expression, cases []*SelectCase) Expression {
	return ctx.annotateExpression(&Select{
		expressionBase: newExpressionBase(loc),
		condition:      condition,
		cases:          cases,
	})
}

func (e *Select) checkPatterns() error {
	if err := e.condition.checkPatterns(); err != nil {
		return err
	}

	if err := checkPatterns(common.Map(func(cs *SelectCase) Pattern { return cs.pattern }, e.cases)); err != nil {
		return err
	}

	for _, cs := range e.cases {
		if err := cs.expression.checkPatterns(); err != nil {
			return err
		}
	}
	return nil
}

func (e *Select) mapTypes(subst map[uint64]Type) error {
	var err error
	e.type_, err = e.type_.mapTo(subst)
	if err != nil {
		return err
	}
	err = e.condition.mapTypes(subst)
	if err != nil {
		return err
	}
	for _, cs := range e.cases {
		err = cs.pattern.mapTypes(subst)
		if err != nil {
			return err
		}
		err = cs.expression.mapTypes(subst)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *Select) Children() []Statement {
	ch := e.expressionBase.Children()
	ch = append(ch, e.condition)
	return append(ch, common.FlatMap(
		func(x *SelectCase) []Statement { return []Statement{x.pattern, x.expression} },
		e.cases)...)
}

func (e *Select) Code(currentModule ast.QualifiedIdentifier) string {
	return fmt.Sprintf("select %s %s end",
		e.condition.Code(currentModule), common.Fold(
			func(x *SelectCase, s string) string {
				if s != "" {
					s += " "
				}
				return s + "case " + x.pattern.Code(currentModule) + " -> " + x.expression.Code(currentModule)
			}, "", e.cases))
}

func (e *Select) appendEquations(eqs Equations, loc *ast.Location, localDefs localTypesMap, ctx *SolvingContext, stack []*Definition) (Equations, error) {
	var err error

	eqs, err = e.condition.appendEquations(eqs, loc, localDefs, ctx, stack)
	if err != nil {
		return nil, err
	}
	for _, cs := range e.cases {
		eqs = append(eqs,
			NewEquation(e, e.condition.Type(), cs.pattern.Type()),
			NewEquation(e, e.type_, cs.expression.Type()))
	}

	for _, cs := range e.cases {
		eqs, err = cs.pattern.appendEquations(eqs, loc, localDefs, ctx, stack)
		if err != nil {
			return nil, err
		}

		eqs, err = cs.expression.appendEquations(eqs, loc, localDefs, ctx, stack)
		if err != nil {
			return nil, err
		}
	}
	return eqs, nil
}

func (e *Select) appendBytecode(ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary) ([]bytecode.Op, []ast.Location) {
	ops, locations = e.condition.appendBytecode(ops, locations, binary)
	var jumpToEndIndices []int
	var prevMatchOpIndex int
	for i, cs := range e.cases {
		if i > 0 {
			//jump to the next case
			ops[prevMatchOpIndex] = ops[prevMatchOpIndex].WithDelta(int32(len(ops) - prevMatchOpIndex - 1))
		}

		ops, locations = cs.pattern.appendBytecode(ops, locations, binary)
		prevMatchOpIndex = len(ops)
		ops, locations = bytecode.AppendMatch(0, cs.location, ops, locations)
		ops, locations = cs.expression.appendBytecode(ops, locations, binary)
		jumpToEndIndices = append(jumpToEndIndices, len(ops))
		ops, locations = bytecode.AppendJump(0, cs.location, ops, locations)
	}

	selectEndIndex := len(ops)
	for _, jumpOpIndex := range jumpToEndIndices {
		//jump to the end
		ops[jumpOpIndex] = ops[jumpOpIndex].WithDelta(int32(selectEndIndex - jumpOpIndex - 1))
	}

	return bytecode.AppendSwapPop(e.location, bytecode.SwapPopModeBoth, ops, locations)
}

type SelectCase struct {
	location   ast.Location
	pattern    Pattern
	expression Expression
}

func NewSelectCase(loc ast.Location, pattern Pattern, expression Expression) *SelectCase {
	return &SelectCase{
		location:   loc,
		pattern:    pattern,
		expression: expression,
	}
}

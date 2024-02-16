package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/bytecode"
	"nar-compiler/internal/pkg/common"
)

type Tuple struct {
	*expressionBase
	items []Expression
}

func NewTuple(ctx *SolvingContext, loc ast.Location, items []Expression) Expression {
	return ctx.annotateExpression(&Tuple{
		expressionBase: newExpressionBase(loc),
		items:          items,
	})
}

func (e Tuple) checkPatterns() error {
	for _, item := range e.items {
		if err := item.checkPatterns(); err != nil {
			return err
		}
	}
	return nil
}

func (e Tuple) mapTypes(subst map[uint64]Type) error {
	var err error
	e.type_, err = e.type_.mapTo(subst)
	if err != nil {
		return err
	}
	for _, item := range e.items {
		err = item.mapTypes(subst)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e Tuple) Children() []Statement {
	return append(e.expressionBase.Children(), common.Map(func(x Expression) Statement { return x }, e.items)...)
}

func (e Tuple) Code(currentModule ast.QualifiedIdentifier) string {
	return fmt.Sprintf("( %s )",
		common.Fold(func(x Expression, s string) string {
			if s != "" {
				s += ", "
			}
			return s + x.Code(currentModule)
		}, "", e.items))
}

func (e Tuple) appendEquations(eqs Equations, loc *ast.Location, localDefs localTypesMap, ctx *SolvingContext, stack []*Definition) (Equations, error) {
	items, err := common.MapError(func(e Expression) (Type, error) {
		itemType := e.Type()
		if itemType == nil {
			return nil, common.NewErrorOf(e, "type cannot be inferred")
		}
		return itemType, nil
	}, e.items)
	if err != nil {
		return nil, err
	}
	eqs = append(eqs, NewEquation(e, e.type_, NewTTuple(e.location, items)))
	for _, item := range e.items {
		eqs, err = item.appendEquations(eqs, loc, localDefs, ctx, stack)
		if err != nil {
			return nil, err
		}
	}
	return eqs, nil
}

func (e Tuple) appendBytecode(ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary) ([]bytecode.Op, []ast.Location) {
	for _, item := range e.items {
		ops, locations = item.appendBytecode(ops, locations, binary)
	}
	return bytecode.AppendMakeObject(bytecode.ObjectKindTuple, len(e.items), e.location, ops, locations)
}

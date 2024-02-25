package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
	bytecode "nar-compiler/pkg/bytecode"
)

type List struct {
	*expressionBase
	items    []Expression
	itemType Type
}

func NewList(ctx *SolvingContext, loc ast.Location, items []Expression) Expression {
	list := &List{
		expressionBase: newExpressionBase(loc),
		items:          items,
	}
	list.itemType = ctx.newTypeAnnotation(list)
	return ctx.annotateExpression(list)
}

func (e *List) checkPatterns() error {
	for _, item := range e.items {
		if err := item.checkPatterns(); err != nil {
			return err
		}
	}
	return nil
}

func (e *List) mapTypes(subst map[uint64]Type) error {
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

func (e *List) Children() []Statement {
	return append(e.expressionBase.Children(), common.Map(func(x Expression) Statement { return x }, e.items)...)
}

func (e *List) Code(currentModule ast.QualifiedIdentifier) string {
	return fmt.Sprintf("[%s]",
		common.Fold(func(x Expression, s string) string {
			if s != "" {
				s += ", "
			}
			return s + x.Code(currentModule)
		}, "", e.items))
}

func (e *List) appendEquations(eqs Equations, loc *ast.Location, localDefs localTypesMap, ctx *SolvingContext, stack []*Definition) (Equations, error) {
	var err error
	for _, item := range e.items {
		eqs = append(eqs, NewEquation(e, e.itemType, item.Type()))
	}

	typeList := NewTNative(e.location, common.NarBaseListList, []Type{e.itemType})
	eqs = append(eqs, NewEquation(e, e.type_, typeList))

	for _, item := range e.items {
		eqs, err = item.appendEquations(eqs, loc, localDefs, ctx, stack)
		if err != nil {
			return nil, err
		}
	}
	return eqs, nil
}

func (e *List) appendBytecode(ops []bytecode.Op, locations []bytecode.Location, binary *bytecode.Binary, hash *bytecode.BinaryHash) ([]bytecode.Op, []bytecode.Location) {
	var err error
	for _, item := range e.items {
		ops, locations = item.appendBytecode(ops, locations, binary, hash)
		if err != nil {
			return nil, nil
		}
	}
	return bytecode.AppendMakeObject(bytecode.ObjectKindList, len(e.items), e.location.Bytecode(), ops, locations)
}

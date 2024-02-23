package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	bytecode "nar-compiler/pkg/bytecode"
)

type Access struct {
	*expressionBase
	fieldName ast.Identifier
	record    Expression
}

func NewAccess(ctx *SolvingContext, loc ast.Location, fieldName ast.Identifier, record Expression) Expression {
	return ctx.annotateExpression(&Access{
		expressionBase: newExpressionBase(loc),
		fieldName:      fieldName,
		record:         record,
	})
}

func (e *Access) checkPatterns() error {
	return e.record.checkPatterns()
}

func (e *Access) mapTypes(subst map[uint64]Type) error {
	var err error
	e.type_, err = e.type_.mapTo(subst)
	if err != nil {
		return err
	}
	return e.record.mapTypes(subst)
}

func (e *Access) Children() []Statement {
	return append(e.expressionBase.Children(), e.record)
}

func (e *Access) Code(currentModule ast.QualifiedIdentifier) string {
	return fmt.Sprintf("%s.%s", e.record.Code(currentModule), e.fieldName)
}

func (e *Access) appendEquations(eqs Equations, loc *ast.Location, localDefs localTypesMap, ctx *SolvingContext, stack []*Definition) (Equations, error) {
	var err error
	fields := map[ast.Identifier]Type{}
	fields[e.fieldName] = e.type_
	eqs = append(eqs, NewEquation(e, NewTRecord(e.location, fields, true), e.record.Type()))
	eqs, err = e.record.appendEquations(eqs, loc, localDefs, ctx, stack)
	if err != nil {
		return nil, err
	}
	return eqs, nil
}

func (e *Access) appendBytecode(ops []bytecode.Op, locations []bytecode.Location, binary *bytecode.Binary) ([]bytecode.Op, []bytecode.Location) {
	var err error
	ops, locations = e.record.appendBytecode(ops, locations, binary)
	if err != nil {
		return nil, nil
	}
	return bytecode.AppendAccess(string(e.fieldName), e.location.Bytecode(), ops, locations, binary)
}

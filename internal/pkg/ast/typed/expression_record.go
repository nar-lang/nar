package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
	"nar-compiler/pkg/bytecode"
)

type Record struct {
	*expressionBase
	fields []*RecordField
}

func NewRecord(ctx *SolvingContext, loc ast.Location, fields []*RecordField) Expression {
	return ctx.annotateExpression(&Record{
		expressionBase: newExpressionBase(loc),
		fields:         fields,
	})
}

func (e *Record) checkPatterns() error {
	for _, field := range e.fields {
		if err := field.value.checkPatterns(); err != nil {
			return err
		}
	}
	return nil
}

func (e *Record) mapTypes(subst map[uint64]Type) error {
	var err error
	e.type_, err = e.type_.mapTo(subst)
	if err != nil {
		return err
	}
	for _, f := range e.fields {
		f.type_, err = f.type_.mapTo(subst)
		if err != nil {
			return err
		}
		err = f.value.mapTypes(subst)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *Record) Children() []Statement {
	return append(e.expressionBase.Children(), common.Map(func(x *RecordField) Statement { return x.value }, e.fields)...)
}

func (e *Record) Code(currentModule ast.QualifiedIdentifier) string {
	return fmt.Sprintf("{%s}",
		common.Fold(func(x *RecordField, s string) string {
			if s != "" {
				s += ", "
			}
			return s + string(x.name) + " = " + x.value.Code(currentModule)
		}, "", e.fields))
}

func (e *Record) appendEquations(eqs Equations, loc *ast.Location, localDefs localTypesMap, ctx *SolvingContext, stack []*Definition) (Equations, error) {
	var err error
	fieldTypes := map[ast.Identifier]Type{}
	for _, f := range e.fields {
		fieldTypes[f.name] = f.type_
	}

	typeRecord := NewTRecord(e.location, fieldTypes, false)
	eqs = append(eqs, NewEquation(e, e.type_, typeRecord))

	for _, f := range e.fields {
		eqs = append(eqs, NewEquation(e, f.type_, f.value.Type()))
	}

	for _, f := range e.fields {
		eqs, err = f.value.appendEquations(eqs, loc, localDefs, ctx, stack)
		if err != nil {
			return nil, err
		}
	}
	return eqs, nil
}

func (e *Record) appendBytecode(ops []bytecode.Op, locations []bytecode.Location, binary *bytecode.Binary) ([]bytecode.Op, []bytecode.Location) {
	var err error
	for _, f := range e.fields {
		ops, locations = f.value.appendBytecode(ops, locations, binary)
		if err != nil {
			return nil, nil
		}
		ops, locations = ast.CString{Value: string(f.name)}.AppendBytecode(
			bytecode.StackKindObject, f.location, ops, locations, binary)
		if err != nil {
			return nil, nil
		}
	}
	return bytecode.AppendMakeObject(bytecode.ObjectKindRecord, len(e.fields), e.location.Bytecode(), ops, locations)
}

type RecordField struct {
	location ast.Location
	type_    Type
	name     ast.Identifier
	value    Expression
}

func (r RecordField) Code(currentModule ast.QualifiedIdentifier) string {
	return fmt.Sprintf("{..., %s = %s, ...}", r.name, r.value.Code(currentModule))
}

func (r RecordField) Location() ast.Location {
	return r.location
}

func NewRecordField(
	ctx *SolvingContext, loc ast.Location, name ast.Identifier, value Expression,
) *RecordField {
	f := &RecordField{
		location: loc,
		name:     name,
		value:    value,
	}
	f.type_ = ctx.newTypeAnnotation(f)
	return f
}

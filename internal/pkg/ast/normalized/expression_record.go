package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
)

type Record struct {
	*expressionBase
	fields []*RecordField
}

func NewRecord(loc ast.Location, fields []*RecordField) Expression {
	return &Record{
		expressionBase: newExpressionBase(loc),
		fields:         fields,
	}
}

func (e *Record) flattenLambdas(parentName ast.Identifier, m *Module, locals map[ast.Identifier]Pattern) Expression {
	for i, a := range e.fields {
		e.fields[i].value = a.value.flattenLambdas(parentName, m, locals)
	}
	return e
}

func (e *Record) replaceLocals(replace map[ast.Identifier]Expression) Expression {
	for i, a := range e.fields {
		e.fields[i].value = a.value.replaceLocals(replace)
	}
	return e
}

func (e *Record) extractUsedLocalsSet(definedLocals map[ast.Identifier]Pattern, usedLocals map[ast.Identifier]struct{}) {
	for _, f := range e.fields {
		f.value.extractUsedLocalsSet(definedLocals, usedLocals)
	}
}

func (e *Record) annotate(ctx *typed.SolvingContext, typeParams typeParamsMap, modules map[ast.QualifiedIdentifier]*Module, typedModules map[ast.QualifiedIdentifier]*typed.Module, moduleName ast.QualifiedIdentifier, stack []*typed.Definition) (typed.Expression, error) {
	fields, err := common.MapError(func(f *RecordField) (*typed.RecordField, error) {
		value, err := f.value.annotate(ctx, typeParams, modules, typedModules, moduleName, stack)
		if err != nil {
			return nil, err
		}
		return typed.NewRecordField(ctx, e.location, f.name, value), nil
	}, e.fields)
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(typed.NewRecord(ctx, e.location, fields))
}

type RecordField struct {
	location ast.Location
	name     ast.Identifier
	value    Expression
}

func NewRecordField(loc ast.Location, name ast.Identifier, value Expression) *RecordField {
	return &RecordField{
		location: loc,
		name:     name,
		value:    value,
	}
}

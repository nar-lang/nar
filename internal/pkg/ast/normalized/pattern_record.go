package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
)

type PRecord struct {
	*patternBase
	fields []*PRecordField
}

func NewPRecord(loc ast.Location, declaredType Type, fields []*PRecordField) Pattern {
	return &PRecord{
		patternBase: newPatternBase(loc, declaredType),
		fields:      fields,
	}
}

func (e *PRecord) extractLocals(locals map[ast.Identifier]Pattern) {
	for _, v := range e.fields {
		locals[v.name] = e
	}
}

func (e *PRecord) annotate(ctx *typed.SolvingContext, typeParams typeParamsMap, modules map[ast.QualifiedIdentifier]*Module, typedModules map[ast.QualifiedIdentifier]*typed.Module, moduleName ast.QualifiedIdentifier, typeMapSource bool, stack []*typed.Definition) (typed.Pattern, error) {
	fields, err := common.MapError(func(f *PRecordField) (*typed.PRecordField, error) {
		return typed.NewPRecordField(ctx, f.location, f.name, nil), nil
	}, e.fields)
	if err != nil {
		return nil, err
	}
	annotatedDeclaredType, err := annotateTypeSafe(ctx, e.declaredType, typeParams, typeMapSource)
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(typed.NewPRecord(ctx, e.location, annotatedDeclaredType, fields))
}

func (e *PRecord) Fields() []*PRecordField {
	return e.fields
}

type PRecordField struct {
	location ast.Location
	name     ast.Identifier
}

func (f PRecordField) Name() ast.Identifier {
	return f.name
}

func NewPRecordField(loc ast.Location, name ast.Identifier) *PRecordField {
	return &PRecordField{
		location: loc,
		name:     name,
	}
}

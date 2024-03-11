package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
	"nar-compiler/pkg/bytecode"
)

type PRecord struct {
	*patternBase
	fields []*PRecordField
}

func NewPRecord(ctx *SolvingContext, loc ast.Location, declaredType Type, fields []*PRecordField) Pattern {
	return ctx.annotatePattern(&PRecord{
		patternBase: newPatternBase(loc, declaredType),
		fields:      fields,
	})
}

func (p *PRecord) simplify() simplePattern {
	return simpleAnything{}
}

func (p *PRecord) mapTypes(subst map[uint64]Type) error {
	var err error
	p.type_, err = p.type_.mapTo(subst)
	if err != nil {
		return err
	}
	for _, f := range p.fields {
		f.type_, err = f.type_.mapTo(subst)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *PRecord) Code(currentModule ast.QualifiedIdentifier) string {
	s := fmt.Sprintf("{%s}",
		common.Fold(func(x *PRecordField, s string) string {
			if s != "" {
				s += ", "
			}
			s += string(x.name)
			if x.type_ != nil {
				s += ": " + x.type_.Code(currentModule)
			}
			return s
		}, "", p.fields))
	if p.declaredType != nil {
		s += ": " + p.declaredType.Code(currentModule)
	}
	return s
}

func (p *PRecord) Children() []Statement {
	return append(p.patternBase.Children(), common.Map(func(x *PRecordField) Statement { return x.type_ }, p.fields)...)
}

func (p *PRecord) appendBytecode(ops []bytecode.Op, locations []bytecode.Location, binary *bytecode.Binary, hash *bytecode.BinaryHash) ([]bytecode.Op, []bytecode.Location) {
	for _, f := range p.fields {
		ops, locations = ast.CString{Value: string(f.name)}.AppendBytecode(bytecode.StackKindPattern, f.location, ops, locations, binary, hash)
	}
	return bytecode.AppendMakePatternLong(bytecode.PatternKindRecord, uint32(len(p.fields)), p.location.Bytecode(), ops, locations, binary)
}

func (p *PRecord) appendEquations(eqs Equations, loc *ast.Location, localDefs localTypesMap, ctx *SolvingContext, stack []*Definition) (Equations, error) {
	fields := map[ast.Identifier]Type{}
	for _, f := range p.fields {
		fields[f.name] = f.type_
	}

	typeRecord := NewTRecord(p.location, fields, true)
	eqs = append(eqs, NewEquation(p, p.type_, typeRecord))

	if p.declaredType != nil {
		eqs = append(eqs, NewEquation(p, p.type_, p.declaredType))
	}
	return eqs, nil
}

type PRecordField struct {
	location     ast.Location
	name         ast.Identifier
	type_        Type
	declaredType Type
}

func (f PRecordField) Code(currentModule ast.QualifiedIdentifier) string {
	return fmt.Sprintf("{..., %s: %s, ...}", f.name, f.type_.Code(currentModule))
}

func (f PRecordField) Location() ast.Location {
	return f.location
}

func NewPRecordField(
	ctx *SolvingContext, loc ast.Location, name ast.Identifier, declaredType Type,
) *PRecordField {
	f := &PRecordField{
		location:     loc,
		name:         name,
		declaredType: declaredType,
	}
	f.type_ = ctx.newTypeAnnotation(f)
	return f
}

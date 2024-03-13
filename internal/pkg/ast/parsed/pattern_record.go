package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
)

func NewPRecord(loc ast.Location, fields []*PRecordField) Pattern {
	return &PRecord{
		patternBase: newPatternBase(loc),
		fields:      fields,
	}
}

type PRecord struct {
	*patternBase
	fields []*PRecordField
}

func (e *PRecord) SemanticTokens() []ast.SemanticToken {
	return []ast.SemanticToken{e.location.ToToken(ast.TokenTypeStruct)}
}

func (e *PRecord) Iterate(f func(statement Statement)) {
	f(e)
	e.patternBase.Iterate(f)
}

type PRecordField struct {
	location ast.Location
	name     ast.Identifier
}

func NewPRecordField(loc ast.Location, name ast.Identifier) *PRecordField {
	return &PRecordField{
		location: loc,
		name:     name,
	}
}

func (e *PRecord) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Pattern, error) {
	var declaredType normalized.Type
	var err error
	if e.declaredType != nil {
		declaredType, err = e.declaredType.normalize(modules, module, nil)
	}
	fields := common.Map(func(x *PRecordField) *normalized.PRecordField {
		locals[x.name] = normalized.NewPNamed(x.location, nil, x.name)
		return normalized.NewPRecordField(x.location, x.name)
	}, e.fields)
	return e.setSuccessor(normalized.NewPRecord(e.location, declaredType, fields)), err
}

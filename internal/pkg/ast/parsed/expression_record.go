package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

func NewRecord(location ast.Location, fields []*RecordField) Expression {
	return &Record{
		expressionBase: newExpressionBase(location),
		fields:         fields,
	}
}

type Record struct {
	*expressionBase
	fields []*RecordField
}

func (e *Record) SemanticTokens() []ast.SemanticToken {
	return []ast.SemanticToken{e.location.ToToken(ast.TokenTypeStruct)}
}

func (e *Record) Iterate(f func(statement Statement)) {
	f(e)
	for _, field := range e.fields {
		if field != nil {
			field.value.Iterate(f)
		}
	}
}

func (e *Record) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	var fields []*normalized.RecordField
	for _, field := range e.fields {
		nValue, err := field.value.normalize(locals, modules, module, normalizedModule)
		if err != nil {
			return nil, err
		}
		fields = append(fields, normalized.NewRecordField(field.location, field.name, nValue))
	}
	return e.setSuccessor(normalized.NewRecord(e.location, fields))
}

type RecordField struct {
	location ast.Location
	name     ast.Identifier
	value    Expression
}

func NewRecordField(location ast.Location, name ast.Identifier, value Expression) *RecordField {
	return &RecordField{
		location: location,
		name:     name,
		value:    value,
	}
}

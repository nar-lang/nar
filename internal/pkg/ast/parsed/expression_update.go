package parsed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
)

func NewUpdate(location ast.Location, recordName ast.QualifiedIdentifier, fields []*RecordField) Expression {
	return &Update{
		expressionBase: newExpressionBase(location),
		recordName:     recordName,
		fields:         fields,
	}
}

type Update struct {
	*expressionBase
	recordName ast.QualifiedIdentifier
	fields     []*RecordField
}

func (e *Update) Iterate(f func(statement Statement)) {
	f(e)
	for _, field := range e.fields {
		if field != nil {
			field.value.Iterate(f)
		}
	}
}

func (e *Update) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	var fields []*normalized.RecordField
	for _, field := range e.fields {
		value, err := field.value.normalize(locals, modules, module, normalizedModule)
		if err != nil {
			return nil, err
		}
		fields = append(fields, normalized.NewRecordField(field.location, field.name, value))
	}

	d, m, ids := module.findDefinitionAndAddDependency(modules, e.recordName, normalizedModule)
	if len(ids) == 1 {
		return normalized.NewUpdateGlobal(e.location, m.name, d.name(), fields), nil
	} else if len(ids) > 1 {
		return nil, newAmbiguousDefinitionError(ids, e.recordName, e.location)
	}

	if lc, ok := locals[ast.Identifier(e.recordName)]; ok {
		return e.setSuccessor(normalized.NewUpdateLocal(e.location, ast.Identifier(e.recordName), lc, fields))
	} else {
		return nil, common.Error{
			Location: e.location,
			Message:  fmt.Sprintf("identifier `%s` not found", e.location.Text()),
		}
	}
}

package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

type Access struct {
	*expressionBase
	record    Expression
	fieldName ast.Identifier
}

func NewAccess(location ast.Location, record Expression, fieldName ast.Identifier) Expression {
	return &Access{
		expressionBase: newExpressionBase(location),
		record:         record,
		fieldName:      fieldName,
	}
}

func (e *Access) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	record, err := e.record.normalize(locals, modules, module, normalizedModule)
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(normalized.NewAccess(e.location, record, e.fieldName))
}

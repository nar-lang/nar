package parsed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
)

func normalizeExpressionSafe(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
	expr Expression) (normalized.Expression, error) {

	if expr == nil {
		return nil, nil
	}
	normalizedExpression, err := expr.normalize(locals, modules, module, normalizedModule)
	if err != nil {
		return nil, err
	}
	expr.setSuccessor(normalizedExpression)
	return normalizedExpression, nil
}

func newAmbiguousInfixError(ids []ast.FullIdentifier, name ast.InfixIdentifier, loc ast.Location) error {
	if len(ids) == 0 {
		return common.Error{
			Location: loc,
			Message:  fmt.Sprintf("infix definition `%s` not found", name),
		}
	} else {
		return common.Error{
			Location: loc,
			Message: fmt.Sprintf(
				"ambiguous infix identifier `%s`, it can be one of %s. Use import to clarify which one to use",
				name, common.Join(ids, ", ")),
		}
	}
}

func newAmbiguousDefinitionError(ids []ast.FullIdentifier, name ast.QualifiedIdentifier, loc ast.Location) error {
	if len(ids) == 0 {
		return common.Error{
			Location: loc,
			Message:  fmt.Sprintf("definition `%s` not found", name),
		}
	} else {
		return common.Error{
			Location: loc,
			Message: fmt.Sprintf(
				"ambiguous identifier `%s`, it can be one of %s. Use import or qualified identifer to clarify which one to use",
				name, common.Join(ids, ", ")),
		}
	}
}

package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

type InfixVar struct {
	*expressionBase
	infix ast.InfixIdentifier
}

func NewInfixVar(location ast.Location, infix ast.InfixIdentifier) Expression {
	return &InfixVar{
		expressionBase: newExpressionBase(location),
		infix:          infix,
	}
}

func (e *InfixVar) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	if i, m, ids := findParsedInfixFn(modules, module, e.infix); len(ids) != 1 {
		return nil, newAmbiguousInfixError(ids, e.infix, e.location)
	} else if d, _, ids := findParsedDefinition(nil, m, ast.QualifiedIdentifier(i.alias), normalizedModule); len(ids) != 1 {
		return nil, newAmbiguousDefinitionError(ids, ast.QualifiedIdentifier(i.alias), e.location)
	} else {
		return e.setSuccessor(normalized.NewGlobal(e.location, m.name, d.name))
	}
}

package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
	"strings"
)

func NewVar(location ast.Location, name ast.QualifiedIdentifier) Expression {
	return &Var{
		expressionBase: newExpressionBase(location),
		name:           name,
	}
}

type Var struct {
	*expressionBase
	name ast.QualifiedIdentifier
}

func (e *Var) SetSuccessor(s normalized.Expression) {
	e.successor = s
}

func (e *Var) Iterate(f func(statement Statement)) {
	f(e)
}

func (e *Var) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	if lc, ok := locals[ast.Identifier(e.name)]; ok {
		return e.setSuccessor(normalized.NewLocal(e.location, ast.Identifier(e.name), lc, e))
	}

	d, m, ids := module.findDefinitionAndAddDependency(modules, e.name, normalizedModule)
	if len(ids) == 1 {
		return e.setSuccessor(normalized.NewGlobal(e.location, m.name, d.Name()))
	} else if len(ids) > 1 {
		return nil, newAmbiguousDefinitionError(ids, e.name, e.location)
	}

	parts := strings.Split(string(e.name), ".")
	if len(parts) > 1 {
		varAccess := NewVar(e.location, ast.QualifiedIdentifier(parts[0]))
		for i := 1; i < len(parts); i++ {
			varAccess = NewAccess(e.location, varAccess, ast.Identifier(parts[i]))
		}
		access, err := varAccess.normalize(locals, modules, module, normalizedModule)
		if err != nil {
			return nil, err
		}
		return e.setSuccessor(access)
	}

	return nil, common.NewErrorOf(e, "identifier `%s` not found", e.location.Text())
}

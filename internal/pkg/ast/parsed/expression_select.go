package parsed

import (
	"maps"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

func NewSelect(location ast.Location, condition Expression, cases []*SelectCase) Expression {
	return &Select{
		expressionBase: newExpressionBase(location),
		condition:      condition,
		cases:          cases,
	}
}

type Select struct {
	*expressionBase
	condition Expression
	cases     []*SelectCase
}

func (e *Select) Iterate(f func(statement Statement)) {
	f(e)
	if e.condition != nil {
		e.condition.Iterate(f)
	}
	for _, cs := range e.cases {
		if cs != nil {
			if cs.pattern != nil {
				cs.pattern.Iterate(f)
			}
			if cs.body != nil {
				cs.body.Iterate(f)
			}
		}
	}
}

func (e *Select) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	condition, err := e.condition.normalize(locals, modules, module, normalizedModule)
	if err != nil {
		return nil, err
	}
	var cases []*normalized.SelectCase
	for _, cs := range e.cases {
		innerLocals := maps.Clone(locals)
		pattern, err := cs.pattern.normalize(innerLocals, modules, module, normalizedModule)
		if err != nil {
			return nil, err
		}
		expression, err := cs.body.normalize(innerLocals, modules, module, normalizedModule)
		if err != nil {
			return nil, err
		}
		cases = append(cases, normalized.NewSelectCase(cs.location, pattern, expression))
	}
	return e.setSuccessor(normalized.NewSelect(e.location, condition, cases))
}

type SelectCase struct {
	location ast.Location
	pattern  Pattern
	body     Expression
}

func NewSelectCase(location ast.Location, pattern Pattern, body Expression) *SelectCase {
	return &SelectCase{
		location: location,
		pattern:  pattern,
		body:     body,
	}
}

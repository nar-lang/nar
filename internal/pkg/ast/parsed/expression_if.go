package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
)

func NewIf(location ast.Location, condition, positive, negative Expression) Expression {
	return &If{
		expressionBase: newExpressionBase(location),
		condition:      condition,
		positive:       positive,
		negative:       negative,
	}
}

type If struct {
	*expressionBase
	condition, positive, negative Expression
}

func (e *If) SemanticTokens() []ast.SemanticToken {
	return nil
}

func (e *If) Iterate(f func(statement Statement)) {
	f(e)
	if e.condition != nil {
		e.condition.Iterate(f)
	}
	if e.positive != nil {
		e.positive.Iterate(f)
	}
	if e.negative != nil {
		e.negative.Iterate(f)
	}
}

func (e *If) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	boolType := normalized.NewTData(
		e.condition.Location(),
		common.NarBaseBasicsBool,
		nil,
		[]*normalized.DataOption{
			normalized.NewDataOption(common.NarTrueName, false, nil),
			normalized.NewDataOption(common.NarFalseName, false, nil),
		},
	)
	condition, err := e.condition.normalize(locals, modules, module, normalizedModule)
	if err != nil {
		return nil, err
	}
	positive, err := e.positive.normalize(locals, modules, module, normalizedModule)
	if err != nil {
		return nil, err
	}
	negative, err := e.negative.normalize(locals, modules, module, normalizedModule)
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(normalized.NewSelect(
		e.location,
		condition,
		[]*normalized.SelectCase{
			normalized.NewSelectCase(
				e.positive.Location(),
				normalized.NewPOption(
					e.positive.Location(), boolType, common.NarBaseBasicsName, common.NarTrueName, nil),
				positive),
			normalized.NewSelectCase(
				e.negative.Location(),
				normalized.NewPOption(
					e.negative.Location(), boolType, common.NarBaseBasicsName, common.NarFalseName, nil),
				negative),
		}))
}

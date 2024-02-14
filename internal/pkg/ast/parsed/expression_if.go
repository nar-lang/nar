package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
)

type If struct {
	*expressionBase
	condition, positive, negative Expression
}

func NewIf(location ast.Location, condition, positive, negative Expression) Expression {
	return &If{
		expressionBase: newExpressionBase(location),
		condition:      condition,
		positive:       positive,
		negative:       negative,
	}
}

func (e *If) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	boolType := normalized.NewTData(
		e.condition.GetLocation(),
		common.NarBaseBasicsBool,
		nil,
		[]*normalized.DataOption{
			normalized.NewDataOption(common.NarBaseBasicsTrueName, false, nil),
			normalized.NewDataOption(common.NarBaseBasicsFalseName, false, nil),
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
				e.positive.GetLocation(),
				normalized.NewPOption(
					e.positive.GetLocation(), boolType, common.NarBaseBasicsName, common.NarBaseBasicsTrueName, nil),
				positive),
			normalized.NewSelectCase(
				e.negative.GetLocation(),
				normalized.NewPOption(
					e.negative.GetLocation(), boolType, common.NarBaseBasicsName, common.NarBaseBasicsFalseName, nil),
				negative),
		}))
}

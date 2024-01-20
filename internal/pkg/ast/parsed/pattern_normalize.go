package parsed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
)

func normalizePattern(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) func(pattern Pattern) (normalized.Pattern, error) {
	return func(pattern Pattern) (normalized.Pattern, error) {
		if pattern == nil {
			return nil, nil
		}
		return pattern.normalize(locals, modules, module, normalizedModule)
	}
}

func (e *PAlias) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Pattern, error) {
	normalize := normalizePattern(locals, modules, module, normalizedModule)
	np := &normalized.PAlias{
		PatternBase: &normalized.PatternBase{Location: e.location},
		Alias:       e.alias,
	}
	locals[e.alias] = np
	var err error
	np.Nested, err = normalize(e.nested)
	if err != nil {
		return nil, err
	}
	np.Type, err = normalizeType(modules, module, nil, nil)(e.type_)
	if err != nil {
		return nil, err
	}
	return np, nil
}

func (e *PAny) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Pattern, error) {
	type_, err := normalizeType(modules, module, nil, nil)(e.type_)
	if err != nil {
		return nil, err
	}
	return &normalized.PAny{
		PatternBase: &normalized.PatternBase{Location: e.location},
		Type:        type_,
	}, nil
}

func (e *PCons) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Pattern, error) {
	normalize := normalizePattern(locals, modules, module, normalizedModule)
	head, err := normalize(e.head)
	if err != nil {
		return nil, err
	}
	tail, err := normalize(e.tail)
	if err != nil {
		return nil, err
	}
	type_, err := normalizeType(modules, module, nil, nil)(e.type_)
	if err != nil {
		return nil, err
	}
	return &normalized.PCons{
		PatternBase: &normalized.PatternBase{Location: e.location},
		Type:        type_,
		Head:        head,
		Tail:        tail,
	}, nil
}

func (e *PConst) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Pattern, error) {
	type_, err := normalizeType(modules, module, nil, nil)(e.type_)
	if err != nil {
		return nil, err
	}
	return &normalized.PConst{
		PatternBase: &normalized.PatternBase{Location: e.location},
		Type:        type_,
		Value:       e.value,
	}, nil
}

func (e *PDataOption) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Pattern, error) {
	normalize := normalizePattern(locals, modules, module, normalizedModule)
	def, mod, ids := findParsedDefinition(modules, module, e.name, normalizedModule)
	if len(ids) == 0 {
		return nil, common.Error{Location: e.location, Message: "data constructor not found"}
	} else if len(ids) > 1 {
		return nil, common.Error{
			Location: e.location,
			Message: fmt.Sprintf(
				"ambiguous data constructor `%s`, it can be one of %s. "+
					"Use import or qualified identifer to clarify which one to use",
				e.name, common.Join(ids, ", ")),
		}
	}
	values, err := common.MapError(normalize, e.values)
	if err != nil {
		return nil, err
	}
	type_, err := normalizeType(modules, module, nil, nil)(e.type_)
	if err != nil {
		return nil, err
	}
	return &normalized.PDataOption{
		PatternBase:    &normalized.PatternBase{Location: e.location},
		Type:           type_,
		ModuleName:     mod.name,
		DefinitionName: def.Name,
		Values:         values,
	}, nil
}

func (e *PList) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Pattern, error) {
	normalize := normalizePattern(locals, modules, module, normalizedModule)
	items, err := common.MapError(normalize, e.items)
	if err != nil {
		return nil, err
	}
	type_, err := normalizeType(modules, module, nil, nil)(e.type_)
	if err != nil {
		return nil, err
	}
	return &normalized.PList{
		PatternBase: &normalized.PatternBase{Location: e.location},
		Type:        type_,
		Items:       items,
	}, nil
}

func (e *PNamed) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Pattern, error) {
	np := &normalized.PNamed{
		PatternBase: &normalized.PatternBase{Location: e.location},
		Name:        e.name,
	}
	locals[e.name] = np
	var err error
	np.Type, err = normalizeType(modules, module, nil, nil)(e.type_)
	if err != nil {
		return nil, err
	}
	return np, nil
}

func (e *PRecord) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Pattern, error) {
	type_, err := normalizeType(modules, module, nil, nil)(e.type_)
	if err != nil {
		return nil, err
	}
	return &normalized.PRecord{
		PatternBase: &normalized.PatternBase{Location: e.location},
		Type:        type_,
		Fields: common.Map(func(x PRecordField) normalized.PRecordField {
			locals[x.name] = &normalized.PNamed{
				PatternBase: &normalized.PatternBase{Location: x.location},
				Name:        x.name,
			}
			return normalized.PRecordField{Location: x.location, Name: x.name}
		}, e.fields),
	}, nil
}

func (e *PTuple) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Pattern, error) {
	normalize := normalizePattern(locals, modules, module, normalizedModule)
	items, err := common.MapError(normalize, e.items)
	if err != nil {
		return nil, err
	}
	type_, err := normalizeType(modules, module, nil, nil)(e.type_)
	if err != nil {
		return nil, err
	}
	return &normalized.PTuple{
		PatternBase: &normalized.PatternBase{Location: e.location},
		Type:        type_,
		Items:       items,
	}, nil
}

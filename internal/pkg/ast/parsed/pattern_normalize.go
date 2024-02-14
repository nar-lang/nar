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
		normalizedPattern, err := pattern.normalize(locals, modules, module, normalizedModule)
		if err != nil {
			return nil, err
		}
		pattern.setSuccessor(normalizedPattern)
		return normalizedPattern, nil
	}
}

func (e *PAlias) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Pattern, error) {
	normalize := normalizePattern(locals, modules, module, normalizedModule)
	nested, err1 := normalize(e.nested)
	type_, err2 := normalizeType(modules, module, nil, nil)(e.type_)
	np := normalized.NewPAlias(e.location, type_, e.alias, nested)
	locals[e.alias] = np
	return np, common.MergeErrors(err1, err2)
}

func (e *PAny) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Pattern, error) {
	type_, err := normalizeType(modules, module, nil, nil)(e.type_)
	return normalized.NewPAny(e.location, type_), err
}

func (e *PCons) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Pattern, error) {
	normalize := normalizePattern(locals, modules, module, normalizedModule)
	head, err1 := normalize(e.head)
	tail, err2 := normalize(e.tail)
	type_, err3 := normalizeType(modules, module, nil, nil)(e.type_)
	return normalized.NewPCons(e.location, type_, head, tail), common.MergeErrors(err1, err2, err3)
}

func (e *PConst) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Pattern, error) {
	type_, err := normalizeType(modules, module, nil, nil)(e.type_)
	return normalized.NewPConst(e.location, type_, e.value), err
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
	values, err1 := common.MapError(normalize, e.values)
	type_, err2 := normalizeType(modules, module, nil, nil)(e.type_)
	return normalized.NewPOption(e.location, type_, mod.name, def.name, values), common.MergeErrors(err1, err2)
}

func (e *PList) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Pattern, error) {
	normalize := normalizePattern(locals, modules, module, normalizedModule)
	items, err1 := common.MapError(normalize, e.items)
	type_, err2 := normalizeType(modules, module, nil, nil)(e.type_)
	return normalized.NewPList(e.location, type_, items), common.MergeErrors(err1, err2)
}

func (e *PNamed) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Pattern, error) {
	type_, err := normalizeType(modules, module, nil, nil)(e.type_)
	np := normalized.NewPNamed(e.location, type_, e.name)
	locals[e.name] = np
	return np, err
}

func (e *PRecord) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Pattern, error) {
	type_, err := normalizeType(modules, module, nil, nil)(e.type_)
	fields := common.Map(func(x PRecordField) *normalized.PRecordField {
		locals[x.name] = normalized.NewPNamed(x.location, nil, x.name)
		return normalized.NewPRecordField(x.location, x.name)
	}, e.fields)
	return normalized.NewPRecord(e.location, type_, fields), err
}

func (e *PTuple) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Pattern, error) {
	normalize := normalizePattern(locals, modules, module, normalizedModule)
	items, err1 := common.MapError(normalize, e.items)
	type_, err2 := normalizeType(modules, module, nil, nil)(e.type_)
	return normalized.NewPTuple(e.location, type_, items), common.MergeErrors(err1, err2)
}

package parsed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
)

type namedTypeMap map[ast.FullIdentifier]*normalized.TPlaceholder

func normalizeType(
	modules map[ast.QualifiedIdentifier]*Module, module *Module, typeModule *Module,
	namedTypes namedTypeMap,
) func(t Type) (normalized.Type, error) {
	return func(t Type) (normalized.Type, error) {
		if t == nil {
			return nil, nil
		}
		p, err := t.normalize(modules, module, typeModule, namedTypes)
		if err != nil {
			return nil, err
		}
		t.setSuccessor(p)
		return p, nil
	}
}

func (t *TFunc) normalize(
	modules map[ast.QualifiedIdentifier]*Module, module *Module, typeModule *Module, namedTypes namedTypeMap,
) (normalized.Type, error) {
	params, err := common.MapError(normalizeType(modules, module, typeModule, namedTypes), t.params)
	if err != nil {
		return nil, err
	}
	ret, err := normalizeType(modules, module, typeModule, namedTypes)(t.return_)
	if err != nil {
		return nil, err
	}
	return normalized.Type(&normalized.TFunc{
		Location: t.location,
		Params:   params,
		Return:   ret,
	}), nil
}

func (t *TRecord) normalize(
	modules map[ast.QualifiedIdentifier]*Module, module *Module, typeModule *Module, namedTypes namedTypeMap,
) (normalized.Type, error) {
	fields := map[ast.Identifier]normalized.Type{}
	for n, v := range t.fields {
		var err error
		fields[n], err = normalizeType(modules, module, typeModule, namedTypes)(v)
		if err != nil {
			return nil, err
		}
	}
	return normalized.Type(&normalized.TRecord{
		Location: t.location,
		Fields:   fields,
	}), nil
}

func (t *TTuple) normalize(
	modules map[ast.QualifiedIdentifier]*Module, module *Module, typeModule *Module, namedTypes namedTypeMap,
) (normalized.Type, error) {
	items, err := common.MapError(normalizeType(modules, module, typeModule, namedTypes), t.items)
	if err != nil {
		return nil, err
	}
	return normalized.Type(&normalized.TTuple{
		Location: t.location,
		Items:    items,
	}), nil
}

func (t *TUnit) normalize(
	modules map[ast.QualifiedIdentifier]*Module, module *Module, typeModule *Module, namedTypes namedTypeMap,
) (normalized.Type, error) {
	return normalized.Type(&normalized.TUnit{
		Location: t.location,
	}), nil
}

func (t *TData) normalize(
	modules map[ast.QualifiedIdentifier]*Module, module *Module, typeModule *Module, namedTypes namedTypeMap,
) (normalized.Type, error) {
	if namedTypes == nil {
		namedTypes = namedTypeMap{}
	}
	if placeholder, cached := namedTypes[t.name]; cached {
		return placeholder, nil
	}
	namedTypes[t.name] = &normalized.TPlaceholder{
		Name: t.name,
	}

	args, err := common.MapError(normalizeType(modules, module, typeModule, namedTypes), t.args)
	if err != nil {
		return nil, err
	}
	options, err := common.MapError(func(x DataOption) (normalized.DataOption, error) {
		values, err := common.MapError(func(x Type) (normalized.Type, error) {
			if typeModule != nil { //TODO: redundant?
				return normalizeType(modules, typeModule, nil, namedTypes)(x)
			} else {
				return normalizeType(modules, module, nil, namedTypes)(x)
			}
		}, x.values)
		if err != nil {
			return normalized.DataOption{}, err
		}
		return normalized.DataOption{
			Name:   x.name,
			Hidden: x.hidden,
			Values: values,
		}, nil
	}, t.options)

	return &normalized.TData{
		Location: t.location,
		Name:     t.name,
		Args:     args,
		Options:  options,
	}, nil
}

func (t *TNative) normalize(
	modules map[ast.QualifiedIdentifier]*Module, module *Module, typeModule *Module, namedTypes namedTypeMap,
) (normalized.Type, error) {
	args, err := common.MapError(normalizeType(modules, module, typeModule, namedTypes), t.args)
	if err != nil {
		return nil, err
	}
	return &normalized.TNative{
		Location: t.location,
		Name:     t.name,
		Args:     args,
	}, nil
}

func (t *TParameter) normalize(
	modules map[ast.QualifiedIdentifier]*Module, module *Module, typeModule *Module, namedTypes namedTypeMap,
) (normalized.Type, error) {
	return &normalized.TTypeParameter{
		Location: t.location,
		Name:     t.name,
	}, nil
}

func (t *TNamed) normalize(
	modules map[ast.QualifiedIdentifier]*Module, module *Module, typeModule *Module, namedTypes namedTypeMap,
) (normalized.Type, error) {
	x, m, ids, err := t.Find(modules, module)
	if err != nil {
		return nil, err
	}
	if ids == nil {
		return nil, common.Error{Location: t.location, Message: fmt.Sprintf("type `%s` not found", t.name)}
	}
	if len(ids) > 1 {
		return nil, common.Error{
			Location: t.location,
			Message: fmt.Sprintf(
				"ambiguous type `%s`, it can be one of %s. Use import or qualified name to clarify which one to use",
				t.name, common.Join(ids, ", ")),
		}
	}
	if named, ok := x.(*TNamed); ok {
		if named.name == t.name {
			return nil, common.Error{
				Location: named.location,
				Message:  fmt.Sprintf("type `%s` aliased to itself", t.name),
			}
		}
	}

	return normalizeType(modules, module, m, namedTypes)(x)
}

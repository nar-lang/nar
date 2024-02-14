package parsed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
	"strings"
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
		normalizedType, err := t.normalize(modules, module, typeModule, namedTypes)
		if err != nil {
			return nil, err
		}
		t.setSuccessor(normalizedType)
		return normalizedType, nil
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
	return normalized.NewTFunc(t.location, params, ret), nil
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
	return normalized.NewTRecord(t.location, fields), nil
}

func (t *TTuple) normalize(
	modules map[ast.QualifiedIdentifier]*Module, module *Module, typeModule *Module, namedTypes namedTypeMap,
) (normalized.Type, error) {
	items, err := common.MapError(normalizeType(modules, module, typeModule, namedTypes), t.items)
	if err != nil {
		return nil, err
	}
	return normalized.NewTTuple(t.location, items), nil
}

func (t *TUnit) normalize(
	modules map[ast.QualifiedIdentifier]*Module, module *Module, typeModule *Module, namedTypes namedTypeMap,
) (normalized.Type, error) {
	return normalized.NewTUnit(t.location), nil
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
	namedTypes[t.name] = normalized.NewTPlaceholder(t.name).(*normalized.TPlaceholder)

	args, err := common.MapError(normalizeType(modules, module, typeModule, namedTypes), t.args)
	if err != nil {
		return nil, err
	}
	options, err := common.MapError(func(x DataOption) (*normalized.DataOption, error) {
		values, err := common.MapError(func(x Type) (normalized.Type, error) {
			if typeModule != nil { //TODO: redundant?
				return normalizeType(modules, typeModule, nil, namedTypes)(x)
			} else {
				return normalizeType(modules, module, nil, namedTypes)(x)
			}
		}, x.values)
		if err != nil {
			return nil, err
		}
		return normalized.NewDataOption(x.name, x.hidden, values), nil
	}, t.options)

	return normalized.NewTData(t.location, t.name, args, options), nil
}

func (t *TNative) normalize(
	modules map[ast.QualifiedIdentifier]*Module, module *Module, typeModule *Module, namedTypes namedTypeMap,
) (normalized.Type, error) {
	args, err := common.MapError(normalizeType(modules, module, typeModule, namedTypes), t.args)
	if err != nil {
		return nil, err
	}
	return normalized.NewTNative(t.location, t.name, args), nil
}

func (t *TParameter) normalize(
	modules map[ast.QualifiedIdentifier]*Module, module *Module, typeModule *Module, namedTypes namedTypeMap,
) (normalized.Type, error) {
	return normalized.NewTParameter(t.location, t.name), nil
}

func (t *TNamed) normalize(
	modules map[ast.QualifiedIdentifier]*Module, module *Module, typeModule *Module, namedTypes namedTypeMap,
) (normalized.Type, error) {
	x, m, ids, err := t.Find(modules, module)
	if err != nil {
		return nil, err
	}
	if ids == nil {
		args := ""
		if len(t.args) > 0 {
			args = fmt.Sprintf("[%s]", strings.Join(common.Repeat("_", len(t.args)), ", "))
		}
		return nil, common.Error{Location: t.location, Message: fmt.Sprintf("type `%s%s` not found", t.name, args)}
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

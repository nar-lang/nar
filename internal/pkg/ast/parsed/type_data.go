package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

type TData struct {
	*typeBase
	name    ast.FullIdentifier
	args    []Type
	options []DataOption
}

func NewTData(loc ast.Location, name ast.FullIdentifier, args []Type, options []DataOption) Type {
	return &TData{
		typeBase: newTypeBase(loc),
		name:     name,
		args:     args,
		options:  options,
	}
}

type DataOption struct {
	name   ast.Identifier
	hidden bool
	values []Type
}

func NewDataOption(name ast.Identifier, hidden bool, values []Type) DataOption {
	return DataOption{
		name:   name,
		hidden: hidden,
		values: values,
	}
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

	var args []normalized.Type
	for _, arg := range t.args {
		nArg, err := arg.normalize(modules, module, typeModule, namedTypes)
		if err != nil {
			return nil, err
		}
		args = append(args, nArg)
	}
	var options []*normalized.DataOption
	for _, option := range t.options {
		var values []normalized.Type
		for _, value := range option.values {
			if typeModule != nil { //TODO: redundant?
				nValue, err := value.normalize(modules, typeModule, typeModule, namedTypes)
				if err != nil {
					return nil, err
				}
				values = append(values, nValue)
			} else {
				nValue, err := value.normalize(modules, module, typeModule, namedTypes)
				if err != nil {
					return nil, err
				}
				values = append(values, nValue)
			}
		}
		options = append(options, normalized.NewDataOption(option.name, option.hidden, values))
	}
	return t.setSuccessor(normalized.NewTData(t.location, t.name, args, options))
}

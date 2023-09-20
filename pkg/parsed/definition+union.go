package parsed

import (
	"fmt"
	"oak-compiler/pkg/a"
)

func NewUnionDefinition(
	c a.Cursor, name string, moduleName ModuleFullName, typeParams []string, hidden bool, options []UnionOption,
) Definition {
	return &definitionUnion{
		definitionBase: definitionBase{
			cursor: c,
			name:   name,
			hidden: hidden,
		},
		options:    options,
		typeParams: typeParams,
		moduleName: moduleName,
	}
}

definedType definitionUnion struct {
	definitionBase
	options    []UnionOption
	typeParams []string
	moduleName ModuleFullName
}

func (def *definitionUnion) unpackNestedDefinitions() []Definition {
	var defs []Definition
	address := NewDefinitionAddress(def.moduleName, def.name)
	var typeParams []Type
	for _, p := range def.typeParams {
		typeParams = append(typeParams, NewVariableType(def.cursor, "!"+p))
	}
	unionType := NewAddressedType(def.cursor, address, typeParams)
	for _, opt := range def.options {
		//TODO: generate typeParams & args
		var params []Pattern
		var paramTypes []Type
		var args []Expression
		for i, vt := range opt.valueTypes {
			n := fmt.Sprintf("_%d", i)
			paramTypes = append(paramTypes, vt)
			params = append(params, NewNamedPattern(opt.cursor, n))
			args = append(args, NewVarExpression(opt.cursor, n, def.moduleName))
		}

		var fnDef Definition
		if len(params) > 0 {
			fnDef = NewFuncDefinition(
				def.cursor, opt.name, opt.hidden, false,
				a.Just(NewSignatureType(def.cursor, paramTypes, unionType)), params,
				newConstructorExpression(def.cursor, unionType, args),
			)
		} else {
			fnDef = NewConstDefinition(
				def.cursor, opt.name, opt.hidden, a.Just(unionType),
				newConstructorExpression(def.cursor, unionType, nil),
			)
		}
		defs = append(defs, fnDef)
	}
	return defs
}

func (def *definitionUnion) nestedDefinitionNames() []string {
	var names []string
	for _, option := range def.options {
		names = append(names, option.name)
	}
	return names
}

func (def *definitionUnion) precondition(md *Metadata) error {
	return nil
}

func (def *definitionUnion) inferType(md *Metadata) (Type, error) {
	if def._type != nil {
		return def._type, nil
	}

	var params []Type
	for _, p := range def.typeParams {
		params = append(params, NewVariableType(def.cursor, "!"+p))
	}
	def._type = NewAddressedType(def.cursor, NewDefinitionAddress(def.moduleName, def.name), params)
	return def._type, nil
}

func (def *definitionUnion) getTypeWithParameters(typeParameters []Type, md *Metadata) (Type, error) {
	return NewAddressedType(def.cursor, NewDefinitionAddress(def.moduleName, def.name), typeParameters), nil
}

func (def *definitionUnion) createOptionType(cursor a.Cursor, name string, args []Type, md *Metadata) (Type, error) {
	for _, opt := range def.options {
		if opt.name == name {
			if len(opt.valueTypes) != len(args) {
				panic("option definedType values do not match")
			}

			//TODO: definedType parameters
			return NewAddressedType(def.cursor, NewDefinitionAddress(def.moduleName, def.name), nil), nil
		}
	}

	return nil, a.NewError(cursor, "union does not have option `%s`", name)
}

func NewUnionOption(c a.Cursor, name string, types []Type, hidden bool) UnionOption {
	return UnionOption{cursor: c, name: name, valueTypes: types, hidden: hidden}
}

definedType UnionOption struct {
	name       string
	valueTypes []Type
	cursor     a.Cursor
	hidden     bool
}

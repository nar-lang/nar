package parsed

import (
	"encoding/json"
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
	"strings"
)

func NewUnionType(c misc.Cursor, modName ModuleFullName, defName string, options []UnionOption) Type {
	return typeUnion{typeBase: typeBase{cursor: c, moduleName: modName}, defName: defName, Options: options}
}

type typeUnion struct {
	TypeUnion__ int
	typeBase
	defName string
	Options []UnionOption
}

func (t typeUnion) extractGenerics(other Type) genericsMap {
	var gm genericsMap
	if to, ok := other.(typeUnion); ok {
		if len(t.Options) == len(to.Options) {
			for i, o := range t.Options {
				gm = mergeGenericMaps(gm, o.valueType.extractGenerics(to.Options[i].valueType))
			}
		}
	}
	return gm
}

func (t typeUnion) equalsTo(other Type, ignoreGenerics bool, md *Metadata) bool {
	o, ok := other.(typeUnion)
	if !ok {
		return false
	}
	if o.moduleName.packageName == t.moduleName.packageName &&
		o.moduleName.moduleName == t.moduleName.moduleName &&
		o.defName == t.defName {
		return true
	}
	return false
}

func (t typeUnion) String() string {
	sb := strings.Builder{}
	for i, x := range t.Options {
		if i > 0 {
			sb.WriteString(" | ")
		}
		sb.WriteString(x.name)
		if _, ok := x.valueType.(typeVoid); !ok {
			sb.WriteString(" ")
			sb.WriteString(x.valueType.String())
		}
	}
	return sb.String()
}

func (t typeUnion) getGenerics() GenericArgs {
	return nil
}

func (t typeUnion) mapGenerics(gm genericsMap) Type {
	var opts []UnionOption
	for _, o := range t.Options {
		o.valueType = o.valueType.mapGenerics(gm)
		opts = append(opts, o)
	}
	t.Options = opts
	return t
}

func (t typeUnion) dereference(md *Metadata) (Type, error) {
	return t, nil
}

func (t typeUnion) nestedDefinitionNames() []string {
	var names []string
	for _, o := range t.Options {
		if !o.hidden {
			names = append(names, o.name)
		}
	}
	return names
}

func (t typeUnion) unpackNestedDefinitions(def Definition) []Definition {
	var defs []Definition
	for _, opt := range t.Options {
		fn := definitionFunc{
			definitionBase: definitionBase{
				Address: DefinitionAddress{
					moduleFullName: def.getAddress().moduleFullName,
					definitionName: opt.name,
				},
				GenericParams: def.getGenerics(),
				Hidden:        def.isHidden(),
				Extern:        false,
			},
		}
		generics := def.getGenerics().toArgs()
		rt := NewAddressedType(t.cursor, t.moduleName, def.getAddress(), generics, false)

		_, noValue := opt.valueType.(typeVoid)
		value := Expression(expressionVoid{cursor: t.cursor})
		if !noValue {
			value = expressionIdentifier{Name: "x"}
		}

		fn.Type = typeSignature{
			Param:      NewNamedParameter(t.cursor, "x"),
			ParamType:  opt.valueType,
			ReturnType: rt,
		}
		fn.Expression = expressionOption{
			Type:     rt,
			Address:  def.getAddress(),
			Generics: def.getGenerics().toArgs(),
			Option:   opt.name,
			Value:    value,
			cursor:   t.cursor,
		}
		defs = append(defs, fn)
	}
	return defs

}

func (t typeUnion) resolveWithRefName(cursor misc.Cursor, refName string, generics GenericArgs, md *Metadata) (resolved.Type, error) {
	resolvedGenerics, err := generics.resolve(cursor, md)
	if err != nil {
		return nil, err
	}
	return resolved.NewRefUnionType(refName, resolvedGenerics), nil
}

func (t typeUnion) resolve(cursor misc.Cursor, md *Metadata) (resolved.Type, error) {
	resolvedOptions, err := t.resolveOptions(md)
	if err != nil {
		return nil, err
	}
	return resolved.NewUnionType(resolvedOptions), nil
}

func (t typeUnion) resolveOptions(md *Metadata) ([]resolved.UnionOption, error) {
	var options []resolved.UnionOption
	for _, opt := range t.Options {
		type_, err := opt.valueType.resolve(opt.cursor, md)
		if err != nil {
			return nil, err
		}
		options = append(options, resolved.NewUnionOption(opt.name, type_))
	}
	return options, nil
}

func NewUnionOption(c misc.Cursor, name string, type_ Type, hidden bool) UnionOption {
	return UnionOption{cursor: c, name: name, valueType: type_, hidden: hidden}
}

type UnionOption struct {
	name      string
	valueType Type
	cursor    misc.Cursor
	hidden    bool
}

func (o UnionOption) Name() string {
	return o.name
}

func (o UnionOption) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name      string
		ValueType Type
	}{
		Name:      o.name,
		ValueType: o.valueType,
	})
}

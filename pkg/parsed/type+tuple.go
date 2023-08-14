package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
	"strings"
)

func NewTupleType(c misc.Cursor, modName ModuleFullName, items []Type) Type {
	return typeTuple{typeBase: typeBase{cursor: c, moduleName: modName}, Items: items}
}

type typeTuple struct {
	typeBase
	Items []Type
}

func (t typeTuple) extractGenerics(other Type) genericsMap {
	var gm genericsMap
	if tt, ok := other.(typeTuple); ok {
		if len(tt.Items) == len(t.Items) {
			for i, item := range t.Items {
				gm = mergeGenericMaps(gm, item.extractGenerics(tt.Items[i]))
			}
		}
	}
	return gm
}

func (t typeTuple) equalsTo(other Type, ignoreGenerics bool, md *Metadata) bool {
	o, ok := other.(typeTuple)
	if !ok {
		return false
	}
	if len(o.Items) != len(t.Items) {
		return false
	}
	for i, x := range t.Items {
		if !typesEqual(x, o.Items[i], ignoreGenerics, md) {
			return false
		}
	}
	return true
}

func (t typeTuple) String() string {
	sb := strings.Builder{}
	sb.WriteString("{")
	for i, x := range t.Items {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(x.String())
	}
	sb.WriteString("}")
	return sb.String()
}

func (t typeTuple) getGenerics() GenericArgs {
	return nil
}

func (t typeTuple) mapGenerics(gm genericsMap) Type {
	var items []Type
	for _, item := range t.Items {
		items = append(items, item.mapGenerics(gm))
	}
	t.Items = items
	return t
}

func (t typeTuple) dereference(md *Metadata) (Type, error) {
	return t, nil
}

func (t typeTuple) nestedDefinitionNames() []string {
	return nil
}

func (t typeTuple) unpackNestedDefinitions(def Definition) []Definition {
	return nil
}

func (t typeTuple) resolveWithRefName(cursor misc.Cursor, refName string, generics GenericArgs, md *Metadata) (resolved.Type, error) {
	resolvedItems, err := t.resolveItems(md)
	if err != nil {
		return nil, err
	}
	resolvedGenerics, err := generics.resolve(cursor, md)
	if err != nil {
		return nil, err
	}
	return resolved.NewRefTupleType(refName, resolvedGenerics, resolvedItems), nil
}

func (t typeTuple) resolve(cursor misc.Cursor, md *Metadata) (resolved.Type, error) {
	resolvedItems, err := t.resolveItems(md)
	if err != nil {
		return nil, err
	}
	return resolved.NewTupleType(resolvedItems), nil
}

func (t typeTuple) resolveItems(md *Metadata) ([]resolved.Type, error) {
	var items []resolved.Type
	for _, item := range t.Items {
		resolvedItem, err := item.resolve(item.getCursor(), md)
		if err != nil {
			return nil, err
		}
		items = append(items, resolvedItem)
	}
	return items, nil
}

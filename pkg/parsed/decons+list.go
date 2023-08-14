package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewListDecons(c misc.Cursor, items []Decons) Decons {
	return deconsList{cursor: c, Items: items}
}

type deconsList struct {
	Items  []Decons
	Alias  string
	cursor misc.Cursor
}

func (d deconsList) SetAlias(alias string) (Decons, error) {
	d.Alias = alias
	return d, nil
}

func (d deconsList) extractLocals(type_ Type, md *Metadata) error {
	if d.Alias != "" {
		md.LocalVars[d.Alias] = type_
	}

	dt, err := type_.dereference(md)
	if err != nil {
		return err
	}
	generics := dt.getGenerics()
	if len(generics) != 1 {
		return misc.NewError(d.cursor, "list type expected one generic argument")
	}
	itemType := generics[0]
	targetType := TypeBuiltinList(d.cursor, md.currentModuleName(), itemType)

	if !typesEqual(targetType, type_, false, md) {
		return misc.NewError(d.cursor, "expected %s got %s", targetType, type_)
	}

	for _, item := range d.Items {
		if err := item.extractLocals(itemType, md); err != nil {
			return err
		}
	}
	return nil
}

func (d deconsList) resolve(type_ Type, md *Metadata) (resolved.Decons, error) {
	if d.Alias != "" {
		md.LocalVars[d.Alias] = type_
	}

	dt, err := type_.dereference(md)
	if err != nil {
		return nil, err
	}

	generics := dt.getGenerics()
	if len(generics) != 1 {
		return nil, misc.NewError(d.cursor, "list type expected one generic argument")
	}
	itemType := generics[0]
	targetType := TypeBuiltinList(d.cursor, md.currentModuleName(), itemType)

	var items []resolved.Decons
	for _, item := range d.Items {
		rd, err := item.resolve(targetType, md)
		if err != nil {
			return nil, err
		}
		items = append(items, rd)
	}

	resolvedItemType, err := itemType.resolve(d.cursor, md)
	if err != nil {
		return nil, err
	}

	return resolved.NewListDecons(resolvedItemType, items, d.Alias), nil
}

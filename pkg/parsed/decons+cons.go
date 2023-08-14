package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewConsDecons(c misc.Cursor, head Decons, tail Decons) Decons {
	return deconsCons{cursor: c, head: head, tail: tail}
}

type deconsCons struct {
	alias  string
	cursor misc.Cursor
	head   Decons
	tail   Decons
}

func (d deconsCons) SetAlias(alias string) (Decons, error) {
	d.alias = alias
	return d, nil
}

func (d deconsCons) extractLocals(type_ Type, md *Metadata) error {
	if d.alias != "" {
		md.LocalVars[d.alias] = type_
	}
	itemType, listType, err := d.getItemAndListType(type_, md)
	if err != nil {
		return err
	}

	err = d.head.extractLocals(itemType, md)
	if err != nil {
		return err
	}

	err = d.tail.extractLocals(listType, md)
	if err != nil {
		return err
	}
	return nil
}

func (d deconsCons) resolve(type_ Type, md *Metadata) (resolved.Decons, error) {
	if d.alias != "" {
		md.LocalVars[d.alias] = type_
	}

	itemType, listType, err := d.getItemAndListType(type_, md)
	if err != nil {
		return nil, err
	}

	resolvedHead, err := d.head.resolve(itemType, md)
	resolvedTail, err := d.tail.resolve(listType, md)

	return resolved.NewConsDecons(resolvedHead, resolvedTail, d.alias), nil
}

func (d deconsCons) getItemAndListType(type_ Type, md *Metadata) (itemType Type, listType Type, err error) {
	dt, err := type_.dereference(md)
	if err != nil {
		return nil, nil, err
	}

	generics := dt.getGenerics()
	if len(generics) != 1 {
		return nil, nil, misc.NewError(d.cursor, "list type expected one generic argument")
	}
	itemType = generics[0]
	listType = TypeBuiltinList(d.cursor, md.currentModuleName(), itemType)
	return
}

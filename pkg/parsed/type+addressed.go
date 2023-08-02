package parsed

import (
	"fmt"
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewAddressedType(
	c misc.Cursor, modName ModuleFullName, address DefinitionAddress, generics GenericArgs, extern bool,
) Type {
	return typeAddressed{
		typeBase: typeBase{cursor: c, moduleName: modName},
		Address:  address,
		Generics: generics,
		Extern:   extern,
	}
}

type typeAddressed struct {
	TypeAddressed__ int
	typeBase
	Address  DefinitionAddress
	Generics GenericArgs
	Extern   bool
}

func (t typeAddressed) extractGenerics(other Type, gm genericsMap) {
	t.Generics.extractGenerics(other.getGenerics(), gm)
}

func (t typeAddressed) equalsTo(other Type, ignoreGenerics bool, md *Metadata) bool {
	o, ok := other.(typeAddressed)

	return ok && o.Address.equalsTo(t.Address) && o.Generics.equalsTo(t.Generics, ignoreGenerics, md)
}

func (t typeAddressed) String() string {
	return fmt.Sprintf(
		"%s:%s.%s%s",
		t.Address.moduleFullName.packageName,
		t.Address.moduleFullName.moduleName,
		t.Address.definitionName,
		t.Generics)
}

func (t typeAddressed) getCursor() misc.Cursor {
	return t.cursor
}

func (t typeAddressed) getGenerics() GenericArgs {
	return t.Generics
}

func (t typeAddressed) mapGenerics(gm genericsMap) Type {
	var gens GenericArgs
	for _, g := range t.Generics {
		gens = append(gens, g.mapGenerics(gm))
	}
	t.Generics = gens
	return t
}

func (t typeAddressed) dereference(md *Metadata) (Type, error) {
	if t.Extern {
		return t, nil
	}

	type_, err := md.getTypeByAddress(t.Address, t.Generics, t.cursor)
	if err != nil {
		return nil, err
	}

	return type_.dereference(md)
}

func (t typeAddressed) nestedDefinitionNames() []string {
	return nil
}

func (t typeAddressed) unpackNestedDefinitions(def Definition) []Definition {
	return nil
}

func (t typeAddressed) resolveWithRefName(cursor misc.Cursor, refName string, generics GenericArgs, md *Metadata) (resolved.Type, error) {
	if t.Extern {
		resolvedGenerics, err := generics.resolve(cursor, md)
		if err != nil {
			return nil, err
		}
		return resolved.NewExternType(refName, resolvedGenerics), nil
	}
	return t.resolve(cursor, md)
}

func (t typeAddressed) resolve(cursor misc.Cursor, md *Metadata) (resolved.Type, error) {
	type_, err := md.getTypeByAddress(t.Address, t.Generics, t.cursor)
	if err != nil {
		return nil, err
	}
	refName, err := md.makeRefNameByAddress(t.Address, t.cursor)
	if err != nil {
		return nil, err
	}
	return type_.resolveWithRefName(cursor, refName, t.Generics, md)
}

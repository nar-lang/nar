package parsed

import (
	"fmt"
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewNamedType(c misc.Cursor, modName ModuleFullName, name string, generics GenericArgs) Type {
	return typeNamed{typeBase: typeBase{cursor: c, moduleName: modName}, Name: name, Generics: generics}
}

type typeNamed struct {
	TypeNamed__ int
	typeBase
	Name     string
	Generics GenericArgs
}

func (t typeNamed) extractGenerics(other Type, gm genericsMap) {
	t.getGenerics().extractGenerics(other.getGenerics(), gm)
}

func (t typeNamed) equalsTo(other Type, ignoreGenerics bool, md *Metadata) bool {
	o, ok := other.(typeNamed)
	return ok && o.Name == t.Name && o.Generics.equalsTo(t.Generics, ignoreGenerics, md)
}

func (t typeNamed) String() string {
	return fmt.Sprintf("%s%s", t.Name, t.Generics)
}

func (t typeNamed) getGenerics() GenericArgs {
	return t.Generics
}

func (t typeNamed) mapGenerics(gm genericsMap) Type {
	var gens GenericArgs
	for _, g := range t.Generics {
		gens = append(gens, g.mapGenerics(gm))
	}
	t.Generics = gens
	return t
}

func (t typeNamed) dereference(md *Metadata) (Type, error) {
	type_, err := md.getTypeByName(t.moduleName, t.Name, t.Generics, t.cursor)
	if err != nil {
		return nil, err
	}

	return type_.dereference(md)
}

func (t typeNamed) nestedDefinitionNames() []string {
	return nil
}

func (t typeNamed) unpackNestedDefinitions(def Definition) []Definition {
	return nil
}

func (t typeNamed) resolveWithRefName(cursor misc.Cursor, refName string, generics GenericArgs, md *Metadata) (resolved.Type, error) {
	return nil, misc.NewError(
		t.cursor, "trying to resolve named type with reference name (this is a compiler error)",
	)
}

func (t typeNamed) resolve(cursor misc.Cursor, md *Metadata) (resolved.Type, error) {
	def, err := md.getTypeByName(t.moduleName, t.Name, t.Generics, t.cursor)
	if err != nil {
		return nil, err
	}
	refName, err := md.makeRefNameByName(t.moduleName, t.Name, t.cursor)
	if err != nil {
		return nil, err
	}
	return def.resolveWithRefName(cursor, refName, t.Generics, md)
}

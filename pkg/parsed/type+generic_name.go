package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewGenericNameType(c misc.Cursor, moduleName ModuleFullName, name string) Type {
	return typeGenericName{typeBase: typeBase{cursor: c, moduleName: moduleName}, Name: name}
}

type typeGenericName struct {
	typeBase
	Name string
}

func (t typeGenericName) extractGenerics(other Type) genericsMap { return nil }

func (t typeGenericName) equalsTo(other Type, ignoreGenerics bool, md *Metadata) bool {
	if ignoreGenerics {
		return true
	}
	o, ok := other.(typeGenericName)
	return ok && o.Name == t.Name
}

func (t typeGenericName) String() string {
	return t.Name
}

func (t typeGenericName) getGenerics() GenericArgs {
	return nil
}

func (t typeGenericName) mapGenerics(gm genericsMap) Type {
	if c, ok := gm[t.Name]; ok {
		return c
	}
	return t
}

func (t typeGenericName) dereference(md *Metadata) (Type, error) {
	return t, nil
}

func (t typeGenericName) nestedDefinitionNames() []string {
	return nil
}

func (t typeGenericName) unpackNestedDefinitions(def Definition) []Definition {
	return nil
}

func (t typeGenericName) resolveWithRefName(cursor misc.Cursor, refName string, generics GenericArgs, md *Metadata) (resolved.Type, error) {
	return nil, misc.NewError(
		t.cursor, "trying to reslve generic nage with reference name (this is a compiler error)",
	)
}

func (t typeGenericName) resolve(cursor misc.Cursor, md *Metadata) (resolved.Type, error) {
	return resolved.NewGenericNameType(t.Name), nil
}

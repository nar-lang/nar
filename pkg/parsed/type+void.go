package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewVoidType(c misc.Cursor, modName ModuleFullName) Type {
	return typeVoid{typeBase: typeBase{cursor: c, moduleName: modName}}
}

type typeVoid struct {
	TypeVoid__ int
	typeBase
}

func (t typeVoid) extractGenerics(other Type, gm genericsMap) {
}

func (t typeVoid) equalsTo(other Type, ignoreGenerics bool, md *Metadata) bool {
	_, ok := other.(typeVoid)
	return ok
}

func (t typeVoid) String() string {
	return "()"
}

func (t typeVoid) getCursor() misc.Cursor {
	return t.cursor
}

func (t typeVoid) getGenerics() GenericArgs {
	return nil
}

func (t typeVoid) mapGenerics(gm genericsMap) Type {
	return t
}

func (t typeVoid) resolve(cursor misc.Cursor, md *Metadata) (resolved.Type, error) {
	return resolved.NewVoidType(), nil
}

func (t typeVoid) dereference(md *Metadata) (Type, error) {
	return t, nil
}

func (t typeVoid) resolveWithRefName(cursor misc.Cursor, refName string, generics GenericArgs, md *Metadata) (resolved.Type, error) {
	return t.resolve(cursor, md)
}

func (t typeVoid) nestedDefinitionNames() []string {
	return nil
}

func (t typeVoid) unpackNestedDefinitions(def Definition) []Definition {
	return nil
}

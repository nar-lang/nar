package parsed

import (
	"fmt"
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

type typeGenericNotResolved struct {
	TypeGenericNotResolved__ int
	typeBase
	Name string
}

func (t typeGenericNotResolved) extractGenerics(other Type, gm genericsMap) {
	gm[t.Name] = other
}

func (t typeGenericNotResolved) equalsTo(other Type, ignoreGenerics bool, md *Metadata) bool {
	if ignoreGenerics {
		return true
	}
	o, ok := other.(typeGenericNotResolved)
	return ok && o.Name == t.Name
}

func (t typeGenericNotResolved) String() string {
	return fmt.Sprintf("(not resolved generic `%s`)", t.Name)
}

func (t typeGenericNotResolved) getGenerics() GenericArgs {
	return nil
}

func (t typeGenericNotResolved) mapGenerics(gm genericsMap) Type {
	if x, ok := gm[t.Name]; ok {
		return x
	}
	return t
}

func (t typeGenericNotResolved) dereference(md *Metadata) (Type, error) {
	return t, nil
}

func (t typeGenericNotResolved) nestedDefinitionNames() []string {
	return nil
}

func (t typeGenericNotResolved) unpackNestedDefinitions(def Definition) []Definition {
	return nil
}

func (t typeGenericNotResolved) resolveWithRefName(
	cursor misc.Cursor, refName string, generics GenericArgs, md *Metadata,
) (resolved.Type, error) {
	return nil, misc.NewError(cursor, "generic type argument cannot be resolved with ref name")
}

func (t typeGenericNotResolved) resolve(cursor misc.Cursor, md *Metadata) (resolved.Type, error) {
	return nil, misc.NewError(cursor, "generic type argument cannot be resolved")
}

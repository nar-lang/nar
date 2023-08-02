package parsed

import (
	"encoding/json"
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

type typeInfix struct {
	TypeInfix__ int
	typeBase
	definition definitionInfix
}

func (t typeInfix) extractGenerics(other Type, gm genericsMap) {

}

func (t typeInfix) equalsTo(other Type, ignoreGenerics bool, md *Metadata) bool {
	o, ok := other.(typeInfix)
	return ok && o.definition.Address.equalsTo(t.definition.Address)
}

func (t typeInfix) String() string {
	return t.definition.Name()
}

func (t typeInfix) getGenerics() GenericArgs {
	return nil
}

func (t typeInfix) mapGenerics(gm genericsMap) Type {
	return t
}

func (t typeInfix) resolve(cursor misc.Cursor, md *Metadata) (resolved.Type, error) {
	dt, err := t.dereference(md)
	if err != nil {
		return nil, err
	}
	return dt.resolve(cursor, md)
}

func (t typeInfix) dereference(md *Metadata) (Type, error) {
	addr := t.definition.Address
	addr.definitionName = t.definition.Alias
	x, err := md.getTypeByAddress(addr, nil, t.cursor)
	if err != nil {
		return nil, err
	}
	return x.dereference(md)
}

func (t typeInfix) resolveWithRefName(cursor misc.Cursor, refName string, generics GenericArgs, md *Metadata) (resolved.Type, error) {
	panic("not supported")
}

func (t typeInfix) nestedDefinitionNames() []string {
	panic("not supported")
}

func (t typeInfix) unpackNestedDefinitions(def Definition) []Definition {
	panic("not supported")
}

func (t typeInfix) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.definition.Name())
}

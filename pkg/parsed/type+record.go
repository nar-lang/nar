package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
	"strings"
)

func NewRecordType(c misc.Cursor, modName ModuleFullName, fields []RecordField) Type {
	return typeRecord{typeBase: typeBase{cursor: c, moduleName: modName}, Fields: fields}
}

type typeRecord struct {
	typeBase
	Fields []RecordField
}

func (t typeRecord) extractGenerics(other Type) genericsMap {
	//TODO implement me
	panic("implement me")
	return nil
}

func (t typeRecord) equalsTo(other Type, ignoreGenerics bool, md *Metadata) bool {
	o, ok := other.(typeRecord)
	if !ok {
		return false
	}
	if len(o.Fields) != len(t.Fields) {
		return false
	}
	for i, f := range t.Fields {
		if !f.equalsTo(o.Fields[i], ignoreGenerics, md) {
			return false
		}
	}

	return true
}

func (t typeRecord) String() string {
	sb := strings.Builder{}
	sb.WriteString("{")
	for i, f := range t.Fields {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(f.name)
		sb.WriteString(":")
		sb.WriteString(f.type_.String())
	}
	sb.WriteString("}")
	return sb.String()
}

func (t typeRecord) getGenerics() GenericArgs {
	return nil
}

func (t typeRecord) mapGenerics(gm genericsMap) Type {
	var fields []RecordField
	for _, f := range t.Fields {
		f.type_ = f.type_.mapGenerics(gm)
		fields = append(fields, f)
	}
	t.Fields = fields
	return t
}

func (t typeRecord) dereference(md *Metadata) (Type, error) {
	return t, nil
}

func (t typeRecord) nestedDefinitionNames() []string {
	return nil
}

func (t typeRecord) unpackNestedDefinitions(def Definition) []Definition {
	return nil
}

func (t typeRecord) resolveWithRefName(cursor misc.Cursor, refName string, generics GenericArgs, md *Metadata) (resolved.Type, error) {
	resolvedGenerics, err := generics.resolve(cursor, md)
	if err != nil {
		return nil, err
	}
	resolvedFields, err := t.resolveFields(md)
	if err != nil {
		return nil, err
	}
	return resolved.NewRefRecordType(refName, resolvedGenerics, resolvedFields), nil
}

func (t typeRecord) resolve(cursor misc.Cursor, md *Metadata) (resolved.Type, error) {
	resolvedFields, err := t.resolveFields(md)
	if err != nil {
		return nil, err
	}
	return resolved.NewRecordType(resolvedFields), nil
}

func (t typeRecord) resolveFields(md *Metadata) ([]resolved.RecordField, error) {
	var fields []resolved.RecordField
	for _, field := range t.Fields {
		resolvedField, err := field.type_.resolve(field.cursor, md)
		if err != nil {
			return nil, err
		}
		fields = append(fields, resolved.NewRecordField(field.name, resolvedField))
	}
	return fields, nil
}

func NewRecordField(c misc.Cursor, name string, type_ Type) RecordField {
	return RecordField{cursor: c, name: name, type_: type_}
}

type RecordField struct {
	name   string
	type_  Type
	cursor misc.Cursor
}

func (f RecordField) Name() string {
	return f.name
}

func (f RecordField) equalsTo(other RecordField, ignoreGenerics bool, md *Metadata) bool {
	return f.name == other.name && typesEqual(f.type_, other.type_, ignoreGenerics, md)
}

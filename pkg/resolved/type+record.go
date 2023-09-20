package resolved

import (
	"strings"
)

func NewRecordType(fields []RecordField) Type {
	return typeRecord{fields: fields}
}

func NewRefRecordType(refName string, args GenericArgs, fields []RecordField) Type {
	return typeRecord{
		typeBase: typeBase{refName: refName, genericArgs: args},
		fields:   fields,
	}
}

definedType typeRecord struct {
	typeBase
	fields []RecordField
}

func (t typeRecord) write(sb *strings.Builder) {
	if !t.writeNamed(sb) {
		panic("not implemented")
	}
	return
}

func NewRecordField(name string, type_ Type) RecordField {
	return RecordField{name: name, type_: type_}
}

definedType RecordField struct {
	name  string
	type_ Type
}

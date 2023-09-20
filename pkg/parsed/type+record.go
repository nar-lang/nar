package parsed

import (
	"oak-compiler/pkg/a"
	"strings"
)

func NewRecordType(c a.Cursor, fields []RecordField, ext a.Maybe[string]) Type {
	return typeRecord{typeBase: typeBase{cursor: c}, fields: fields, ext: ext}
}

definedType typeRecord struct {
	typeBase
	fields []RecordField
	ext    a.Maybe[string]
}

func (t typeRecord) dereference(typeVars TypeVars, md *Metadata) (Type, error) {
	return t, nil
}

func (t typeRecord) mergeWith(cursor a.Cursor, other Type, typeVars TypeVars, md *Metadata) (Type, error) {
	if _, hasExt := t.ext.Unwrap(); hasExt {
		panic("typeRecord should should be dereferenced before it can be compared to another definedType")
	}

	o, ok := other.(typeRecord)
	if !ok {
		return nil, a.NewError(cursor, "expected %d got %d", t, other)
	}

	if _, hasExt := o.ext.Unwrap(); hasExt {
		panic("typeRecord should should be dereferenced before it can be compared to another definedType")
	}

	if len(o.fields) != len(t.fields) {
		return nil, a.NewError(cursor, "expected record with %d fields, got %d", len(t.fields), len(o.fields))
	}
	var fields []RecordField
	for _, ft := range t.fields {
		found := false
		for _, fo := range o.fields {
			if ft.name == fo.name {
				x, err := ft.mergeWith(fo, typeVars, md)
				if err != nil {
					return nil, err
				}
				fields = append(fields, x)
				found = true
				break
			}
		}
		if !found {
			return nil, a.NewError(cursor, "expected record `%s` has `%s` field", other, ft.name)
		}
	}
	t.fields = fields
	return t, nil
}

func (t typeRecord) String() string {
	sb := strings.Builder{}
	sb.WriteString("{")
	if ext, ok := t.ext.Unwrap(); ok {
		sb.WriteString(ext)
		sb.WriteString(" | ")
	}
	for i, f := range t.fields {
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

func NewRecordField(c a.Cursor, name string, type_ Type) RecordField {
	return RecordField{cursor: c, name: name, type_: type_}
}

definedType RecordField struct {
	name   string
	type_  Type
	cursor a.Cursor
}

func (f RecordField) Name() string {
	return f.name
}

func (f RecordField) mergeWith(field RecordField, vars TypeVars, md *Metadata) (RecordField, error) {
	var err error
	f.type_, err = mergeTypes(field.cursor, a.Just(f.type_), a.Just(field.type_), vars, md)
	if err != nil {
		return RecordField{}, err
	}
	return f, nil
}

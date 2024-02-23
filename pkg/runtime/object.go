package runtime

import (
	"fmt"
	"nar-compiler/pkg/bytecode"
	"unsafe"
)

const kTrue = "Nar.Base.Basics.Bool#True"
const kFalse = "Nar.Base.Basics.Bool#False"

var invalidObject = Object{}
var unitObject = Object{index: -1}

type Object struct {
	arena typedArena
	index int
}

func (o Object) Kind() InstanceKind {
	return o.arena.kind()
}

func (o Object) Unwrap() (any, error) {
	switch o.arena.kind() {
	case InstanceKindUnit:
		return o.AsUnit()
	case InstanceKindChar:
		return o.AsChar()
	case InstanceKindInt:
		return o.AsInt()
	case InstanceKindFloat:
		return o.AsFloat()
	case InstanceKindString:
		return o.AsString()
	case InstanceKindRecord:
		return o.AsRecord()
	case InstanceKindTuple:
		return o.AsTuple()
	case InstanceKindList:
		return o.AsList()
	case InstanceKindOption:
		return o.AsOption()
	default:
		return nil, fmt.Errorf("unknown instance kind: %d", o.arena.kind())
	}
}

func (o Object) AsUnit() (struct{}, error) {
	if o.Kind() != InstanceKindUnit {
		return struct{}{}, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindUnit), kindToString(o.Kind()))
	}
	return struct{}{}, nil
}

func (o Object) AsChar() (rune, error) {
	if o.Kind() != InstanceKindChar {
		return 0, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindChar), kindToString(o.Kind()))
	}
	return getObjectValue[rune](o)
}

func (o Object) AsInt() (int64, error) {
	if o.Kind() != InstanceKindInt {
		return 0, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindInt), kindToString(o.Kind()))
	}
	return getObjectValue[int64](o)
}

func (o Object) AsFloat() (float64, error) {
	if o.Kind() != InstanceKindFloat {
		return 0, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindFloat), kindToString(o.Kind()))
	}
	return getObjectValue[float64](o)
}

func (o Object) AsString() (string, error) {
	if o.Kind() != InstanceKindString {
		return "", fmt.Errorf("expected %s, got %s", kindToString(InstanceKindString), kindToString(o.Kind()))
	}
	return getObjectValue[string](o)
}

func (o Object) AsRecord() (map[any]any, error) {
	if o.Kind() != InstanceKindRecord {
		return nil, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindRecord), kindToString(o.Kind()))
	}

	r := make(map[any]any)

	it := o
	for it.index != -1 {
		f, err := getObjectValue[recordField](o)
		if err != nil {
			return nil, fmt.Errorf("failed to get record field: %w", err)
		}
		k, err := f.key.Unwrap()
		if err != nil {
			return nil, fmt.Errorf("failed to unwrap key: %w", err)
		}
		v, err := f.value.Unwrap()
		if err != nil {
			return nil, fmt.Errorf("failed to unwrap value: %w", err)
		}
		r[k] = v
		it.index = f.parent
	}

	return r, nil
}

func (o Object) FindField(name string) (Object, bool, error) {
	if o.Kind() != InstanceKindRecord {
		return invalidObject, false, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindRecord), kindToString(o.Kind()))
	}
	it := o
	for it.index != -1 {
		f, err := getObjectValue[recordField](o)
		if err != nil {
			return invalidObject, false, fmt.Errorf("failed to get record field: %w", err)
		}
		if f.key.index == -1 {
			return invalidObject, false, fmt.Errorf("field %q not found", name)
		}
		key, err := f.key.AsString()
		if err != nil {
			return invalidObject, false, fmt.Errorf("failed to unwrap key: %w", err)
		}
		if key == name {
			return f.value, true, nil
		}
		it.index = f.parent
	}
	return invalidObject, false, nil
}

func (o Object) UpdateField(key Object, value Object) (Object, error) {
	if o.Kind() != InstanceKindRecord {
		return invalidObject, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindRecord), kindToString(o.Kind()))
	}
	if key.Kind() != InstanceKindString {
		return invalidObject, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindString), kindToString(key.Kind()))
	}
	return o.arena.add(recordField{key: key, value: value, parent: o.index}), nil
}

func UnwrapRecordFields[K any, V any](o Object) ([]struct {
	k K
	v V
}, error) {
	if o.Kind() != InstanceKindRecord {
		return nil, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindRecord), kindToString(o.Kind()))
	}
	var r []struct {
		k K
		v V
	}
	it := o
	for it.index != -1 {
		f, err := getObjectValue[recordField](o)
		if err != nil {
			return nil, fmt.Errorf("failed to get record field: %w", err)
		}
		ak, err := f.key.Unwrap()
		if err != nil {
			return nil, fmt.Errorf("failed to unwrap key: %w", err)
		}
		k, ok := ak.(K)
		if !ok {
			return nil, fmt.Errorf("failed to cast key to %T", k)
		}
		av, err := f.value.Unwrap()
		if err != nil {
			return nil, fmt.Errorf("failed to unwrap value: %w", err)
		}
		v, ok := av.(V)
		if !ok {
			return nil, fmt.Errorf("failed to cast value to %T", v)
		}
		r = append(r, struct {
			k K
			v V
		}{k: k, v: v})
		it.index = f.parent
	}
	return r, nil
}

func UnwrapRecord[K comparable, V any](o Object) (map[K]V, error) {
	if o.Kind() != InstanceKindRecord {
		return nil, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindRecord), kindToString(o.Kind()))
	}
	r := make(map[K]V)

	it := o
	for it.index != -1 {
		f, err := getObjectValue[recordField](o)
		if err != nil {
			return nil, fmt.Errorf("failed to get record field: %w", err)
		}
		ak, err := f.key.Unwrap()
		if err != nil {
			return nil, fmt.Errorf("failed to unwrap key: %w", err)
		}
		k, ok := ak.(K)
		if !ok {
			return nil, fmt.Errorf("failed to cast key to %T", k)
		}
		av, err := f.value.Unwrap()
		if err != nil {
			return nil, fmt.Errorf("failed to unwrap value: %w", err)
		}
		v, ok := av.(V)
		if !ok {
			return nil, fmt.Errorf("failed to cast value to %T", v)
		}
		r[k] = v
		it.index = f.parent
	}
	return r, nil
}

func (o Object) AsTuple() ([]any, error) {
	if o.Kind() != InstanceKindTuple {
		return nil, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindTuple), kindToString(o.Kind()))
	}
	var r []any
	it := o
	for it.index != -1 {
		r = append(r)
		f, err := getObjectValue[tupleItem](o)
		if err != nil {
			return nil, fmt.Errorf("failed to get tuple item: %w", err)
		}
		v, err := f.value.Unwrap()
		if err != nil {
			return nil, fmt.Errorf("failed to unwrap value: %w", err)
		}
		r = append(r, v)
		it.index = f.next
	}
	return r, nil
}

func (o Object) AsObjectTuple() ([]Object, error) {
	if o.Kind() != InstanceKindTuple {
		return nil, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindTuple), kindToString(o.Kind()))
	}
	var r []Object
	it := o
	for it.index != -1 {
		f, err := getObjectValue[tupleItem](o)
		if err != nil {
			return nil, fmt.Errorf("failed to get tuple item: %w", err)
		}
		r = append(r, f.value)
		it.index = f.next
	}
	return r, nil

}

func UnwrapTuple[T1, T2 any](o Object) (T1, T2, error) {
	var t1 T1
	var t2 T2
	if o.Kind() != InstanceKindTuple {
		return t1, t2, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindTuple), kindToString(o.Kind()))
	}
	var ok bool

	f1, err := getObjectValue[tupleItem](o)
	if err != nil {
		return t1, t2, fmt.Errorf("failed to get tuple value #1: %w", err)
	}
	av1, err := f1.value.Unwrap()
	if err != nil {
		return t1, t2, fmt.Errorf("failed to unwrap tuple value #1: %w", err)
	}
	t1, ok = av1.(T1)
	if !ok {
		return t1, t2, fmt.Errorf("failed to cast tuple value #1 to %T", t1)
	}
	o.index = f1.next
	f2, err := getObjectValue[tupleItem](o)
	if err != nil {
		return t1, t2, fmt.Errorf("failed to get tuple value #2: %w", err)
	}
	av2, err := f2.value.Unwrap()
	if err != nil {
		return t1, t2, fmt.Errorf("failed to unwrap tuple value #2: %w", err)
	}
	t2, ok = av2.(T2)
	if !ok {
		return t1, t2, fmt.Errorf("failed to cast tuple value #2 to %T", t2)
	}
	return t1, t2, nil
}

func UnwrapTriple[T1, T2, T3 any](o Object) (T1, T2, T3, error) {
	var t1 T1
	var t2 T2
	var t3 T3
	if o.Kind() != InstanceKindTuple {
		return t1, t2, t3, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindTuple), kindToString(o.Kind()))
	}
	var ok bool

	f1, err := getObjectValue[tupleItem](o)
	if err != nil {
		return t1, t2, t3, fmt.Errorf("failed to get tuple value #1: %w", err)
	}
	av1, err := f1.value.Unwrap()
	if err != nil {
		return t1, t2, t3, fmt.Errorf("failed to unwrap tuple value #1: %w", err)
	}
	t1, ok = av1.(T1)
	if !ok {
		return t1, t2, t3, fmt.Errorf("failed to cast tuple value #1 to %T", t1)
	}
	o.index = f1.next
	f2, err := getObjectValue[tupleItem](o)
	if err != nil {
		return t1, t2, t3, fmt.Errorf("failed to get tuple value #2: %w", err)
	}
	av2, err := f2.value.Unwrap()
	if err != nil {
		return t1, t2, t3, fmt.Errorf("failed to unwrap tuple value #2: %w", err)
	}
	t2, ok = av2.(T2)
	if !ok {
		return t1, t2, t3, fmt.Errorf("failed to cast tuple value #2 to %T", t2)
	}
	o.index = f2.next
	f3, err := getObjectValue[tupleItem](o)
	if err != nil {
		return t1, t2, t3, fmt.Errorf("failed to get tuple value #3: %w", err)
	}
	av3, err := f3.value.Unwrap()
	if err != nil {
		return t1, t2, t3, fmt.Errorf("failed to unwrap tuple value #3: %w", err)
	}
	t3, ok = av3.(T3)
	if !ok {
		return t1, t2, t3, fmt.Errorf("failed to cast tuple value #3 to %T", t3)
	}
	return t1, t2, t3, nil
}

func (o Object) AsList() ([]any, error) {
	if o.Kind() != InstanceKindList {
		return nil, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindList), kindToString(o.Kind()))
	}
	var r []any
	it := o
	index := 0
	for it.index != -1 {
		r = append(r)
		f, err := getObjectValue[listItem](o)
		if err != nil {
			return nil, fmt.Errorf("failed to get list item #%d: %w", index, err)
		}
		v, err := f.value.Unwrap()
		if err != nil {
			return nil, fmt.Errorf("failed to unwrap list item #%d: %w", index, err)
		}
		r = append(r, v)
		it.index = f.next
		index++
	}
	return r, nil
}

func (o Object) AsObjectList() ([]Object, error) {
	if o.Kind() != InstanceKindList {
		return nil, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindList), kindToString(o.Kind()))
	}
	var r []Object
	it := o
	for it.index != -1 {
		f, err := getObjectValue[listItem](o)
		if err != nil {
			return nil, fmt.Errorf("failed to get list item: %w", err)
		}
		r = append(r, f.value)
		it.index = f.next
	}
	return r, nil
}

func UnwrapList[T any](o Object) ([]T, error) {
	if o.Kind() != InstanceKindList {
		return nil, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindList), kindToString(o.Kind()))
	}
	var r []T
	it := o
	index := 0
	for it.index != -1 {
		f, err := getObjectValue[listItem](o)
		if err != nil {
			return nil, fmt.Errorf("failed to get list item #%d: %w", index, err)
		}
		av, err := f.value.Unwrap()
		if err != nil {
			return nil, fmt.Errorf("failed to unwrap list item #%d: %w", index, err)
		}
		v, ok := av.(T)
		if !ok {
			return nil, fmt.Errorf("failed to cast list item #%d to %T", index, v)
		}
		r = append(r, v)
		it.index = f.next
		index++
	}
	return r, nil
}

func (o Object) AsOption() (struct {
	name   string
	values []any
}, error) {
	var t struct {
		name   string
		values []any
	}
	if o.Kind() != InstanceKindOption {
		return t, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindOption), kindToString(o.Kind()))
	}
	opt, err := getObjectValue[option](o)
	if err != nil {
		return t, fmt.Errorf("failed to get option: %w", err)
	}
	v, err := opt.values.AsTuple()
	if err != nil {
		return t, fmt.Errorf("failed to unwrap option values: %w", err)
	}
	t.name, err = opt.fullName.AsString()
	if err != nil {
		return t, fmt.Errorf("failed to unwrap option name: %w", err)
	}
	t.values = v
	return t, nil
}

func (o Object) asObjectOption() (Object, Object, error) {
	if o.Kind() != InstanceKindOption {
		return Object{}, Object{}, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindOption), kindToString(o.Kind()))
	}
	opt, err := getObjectValue[option](o)
	if err != nil {
		return Object{}, Object{}, fmt.Errorf("failed to get option: %w", err)
	}
	return opt.fullName, opt.values, nil
}

func UnwrapOption(o Object) (string, error) {
	if o.Kind() != InstanceKindOption {
		return "", fmt.Errorf("expected %s, got %s", kindToString(InstanceKindOption), kindToString(o.Kind()))
	}
	opt, err := getObjectValue[option](o)
	if err != nil {
		return "", fmt.Errorf("failed to get option: %w", err)
	}
	return opt.fullName.AsString()
}

func UnwrapOption1[T1 any](o Object) (string, T1, error) {
	var v1 T1
	var ok bool
	if o.Kind() != InstanceKindOption {
		return "", v1, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindOption), kindToString(o.Kind()))
	}
	opt, err := getObjectValue[option](o)
	if err != nil {
		return "", v1, fmt.Errorf("failed to get option: %w", err)
	}
	v, err := opt.values.AsTuple()
	if err != nil {
		return "", v1, fmt.Errorf("failed to unwrap option values: %w", err)
	}
	if len(v) != 1 {
		return "", v1, fmt.Errorf("expected 1 value in option, got %d", len(v))
	}
	v1, ok = v[0].(T1)
	if !ok {
		return "", v1, fmt.Errorf("failed to cast option value to %T", v1)
	}
	name, err := opt.fullName.AsString()
	if err != nil {
		return "", v1, fmt.Errorf("failed to unwrap option name: %w", err)
	}
	return name, v1, nil
}

func UnwrapOption2[T1, T2 any](o Object) (string, T1, T2, error) {
	var v1 T1
	var v2 T2
	var ok bool
	if o.Kind() != InstanceKindOption {
		return "", v1, v2, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindOption), kindToString(o.Kind()))
	}
	opt, err := getObjectValue[option](o)
	if err != nil {
		return "", v1, v2, fmt.Errorf("failed to get option: %w", err)
	}
	v, err := opt.values.AsTuple()
	if err != nil {
		return "", v1, v2, fmt.Errorf("failed to unwrap option values: %w", err)
	}
	if len(v) != 2 {
		return "", v1, v2, fmt.Errorf("expected 2 values in option, got %d", len(v))
	}
	v1, ok = v[0].(T1)
	if !ok {
		return "", v1, v2, fmt.Errorf("failed to cast option value #1 to %T", v1)
	}
	v2, ok = v[1].(T2)
	if !ok {
		return "", v1, v2, fmt.Errorf("failed to cast option value #2 to %T", v2)
	}
	name, err := opt.fullName.AsString()
	if err != nil {
		return "", v1, v2, fmt.Errorf("failed to unwrap option name: %w", err)
	}
	return name, v1, v2, nil
}

func UnwrapOption3[T1, T2, T3 any](o Object) (string, T1, T2, T3, error) {
	var v1 T1
	var v2 T2
	var v3 T3
	var ok bool
	if o.Kind() != InstanceKindOption {
		return "", v1, v2, v3, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindOption), kindToString(o.Kind()))
	}
	opt, err := getObjectValue[option](o)
	if err != nil {
		return "", v1, v2, v3, fmt.Errorf("failed to get option: %w", err)
	}
	v, err := opt.values.AsTuple()
	if err != nil {
		return "", v1, v2, v3, fmt.Errorf("failed to unwrap option values: %w", err)
	}
	if len(v) != 3 {
		return "", v1, v2, v3, fmt.Errorf("expected 3 values in option, got %d", len(v))
	}
	v1, ok = v[0].(T1)
	if !ok {
		return "", v1, v2, v3, fmt.Errorf("failed to cast option value #1 to %T", v1)
	}
	v2, ok = v[1].(T2)
	if !ok {
		return "", v1, v2, v3, fmt.Errorf("failed to cast option value #2 to %T", v2)
	}
	v3, ok = v[2].(T3)
	if !ok {
		return "", v1, v2, v3, fmt.Errorf("failed to cast option value #3 to %T", v3)
	}
	name, err := opt.fullName.AsString()
	if err != nil {
		return "", v1, v2, v3, fmt.Errorf("failed to unwrap option name: %w", err)
	}
	return name, v1, v2, v3, nil
}

func UnwrapOption4[T1, T2, T3, T4 any](o Object) (string, T1, T2, T3, T4, error) {
	var v1 T1
	var v2 T2
	var v3 T3
	var v4 T4
	var ok bool
	if o.Kind() != InstanceKindOption {
		return "", v1, v2, v3, v4, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindOption), kindToString(o.Kind()))
	}
	opt, err := getObjectValue[option](o)
	if err != nil {
		return "", v1, v2, v3, v4, fmt.Errorf("failed to get option: %w", err)
	}
	v, err := opt.values.AsTuple()
	if err != nil {
		return "", v1, v2, v3, v4, fmt.Errorf("failed to unwrap option values: %w", err)
	}
	if len(v) != 4 {
		return "", v1, v2, v3, v4, fmt.Errorf("expected 4 values in option, got %d", len(v))
	}
	v1, ok = v[0].(T1)
	if !ok {
		return "", v1, v2, v3, v4, fmt.Errorf("failed to cast option value #1 to %T", v1)
	}
	v2, ok = v[1].(T2)
	if !ok {
		return "", v1, v2, v3, v4, fmt.Errorf("failed to cast option value #2 to %T", v2)
	}
	v3, ok = v[2].(T3)
	if !ok {
		return "", v1, v2, v3, v4, fmt.Errorf("failed to cast option value #3 to %T", v3)
	}
	v4, ok = v[3].(T4)
	if !ok {
		return "", v1, v2, v3, v4, fmt.Errorf("failed to cast option value #4 to %T", v4)
	}
	name, err := opt.fullName.AsString()
	if err != nil {
		return "", v1, v2, v3, v4, fmt.Errorf("failed to unwrap option name: %w", err)
	}
	return name, v1, v2, v3, v4, nil
}

func UnwrapOption5[T1, T2, T3, T4, T5 any](o Object) (string, T1, T2, T3, T4, T5, error) {
	var v1 T1
	var v2 T2
	var v3 T3
	var v4 T4
	var v5 T5
	var ok bool
	if o.Kind() != InstanceKindOption {
		return "", v1, v2, v3, v4, v5, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindOption), kindToString(o.Kind()))
	}
	opt, err := getObjectValue[option](o)
	if err != nil {
		return "", v1, v2, v3, v4, v5, fmt.Errorf("failed to get option: %w", err)
	}
	v, err := opt.values.AsTuple()
	if err != nil {
		return "", v1, v2, v3, v4, v5, fmt.Errorf("failed to unwrap option values: %w", err)
	}
	if len(v) != 5 {
		return "", v1, v2, v3, v4, v5, fmt.Errorf("expected 5 values in option, got %d", len(v))
	}
	v1, ok = v[0].(T1)
	if !ok {
		return "", v1, v2, v3, v4, v5, fmt.Errorf("failed to cast option value #1 to %T", v1)
	}
	v2, ok = v[1].(T2)
	if !ok {
		return "", v1, v2, v3, v4, v5, fmt.Errorf("failed to cast option value #2 to %T", v2)
	}
	v3, ok = v[2].(T3)
	if !ok {
		return "", v1, v2, v3, v4, v5, fmt.Errorf("failed to cast option value #3 to %T", v3)
	}
	v4, ok = v[3].(T4)
	if !ok {
		return "", v1, v2, v3, v4, v5, fmt.Errorf("failed to cast option value #4 to %T", v4)
	}
	v5, ok = v[4].(T5)
	if !ok {
		return "", v1, v2, v3, v4, v5, fmt.Errorf("failed to cast option value #5 to %T", v5)
	}
	name, err := opt.fullName.AsString()
	if err != nil {
		return "", v1, v2, v3, v4, v5, fmt.Errorf("failed to unwrap option name: %w", err)
	}
	return name, v1, v2, v3, v4, v5, nil
}

func UnwrapOption6[T1, T2, T3, T4, T5, T6 any](o Object) (string, T1, T2, T3, T4, T5, T6, error) {
	var v1 T1
	var v2 T2
	var v3 T3
	var v4 T4
	var v5 T5
	var v6 T6
	var ok bool
	if o.Kind() != InstanceKindOption {
		return "", v1, v2, v3, v4, v5, v6, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindOption), kindToString(o.Kind()))
	}
	opt, err := getObjectValue[option](o)
	if err != nil {
		return "", v1, v2, v3, v4, v5, v6, fmt.Errorf("failed to get option: %w", err)
	}
	v, err := opt.values.AsTuple()
	if err != nil {
		return "", v1, v2, v3, v4, v5, v6, fmt.Errorf("failed to unwrap option values: %w", err)
	}
	if len(v) != 6 {
		return "", v1, v2, v3, v4, v5, v6, fmt.Errorf("expected 6 values in option, got %d", len(v))
	}
	v1, ok = v[0].(T1)
	if !ok {
		return "", v1, v2, v3, v4, v5, v6, fmt.Errorf("failed to cast option value #1 to %T", v1)
	}
	v2, ok = v[1].(T2)
	if !ok {
		return "", v1, v2, v3, v4, v5, v6, fmt.Errorf("failed to cast option value #2 to %T", v2)
	}
	v3, ok = v[2].(T3)
	if !ok {
		return "", v1, v2, v3, v4, v5, v6, fmt.Errorf("failed to cast option value #3 to %T", v3)
	}
	v4, ok = v[3].(T4)
	if !ok {
		return "", v1, v2, v3, v4, v5, v6, fmt.Errorf("failed to cast option value #4 to %T", v4)
	}
	v5, ok = v[4].(T5)
	if !ok {
		return "", v1, v2, v3, v4, v5, v6, fmt.Errorf("failed to cast option value #5 to %T", v5)
	}
	v6, ok = v[5].(T6)
	if !ok {
		return "", v1, v2, v3, v4, v5, v6, fmt.Errorf("failed to cast option value #6 to %T", v6)
	}
	name, err := opt.fullName.AsString()
	if err != nil {
		return "", v1, v2, v3, v4, v5, v6, fmt.Errorf("failed to unwrap option name: %w", err)
	}
	return name, v1, v2, v3, v4, v5, v6, nil
}

func (o Object) AsBool() (bool, error) {
	if o.Kind() != InstanceKindOption {
		return false, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindOption), kindToString(o.Kind()))
	}
	if o.index == 1 {
		return true, nil

	}
	if o.index == 0 {
		return false, nil
	}
	return false, fmt.Errorf("expected Boolean type")
}

func (o Object) AsFunction0() (func() Object, error) {
	if o.Kind() != InstanceKindFunction {
		return nil, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindFunction), kindToString(o.Kind()))
	}
	fn, err := getObjectValue[function](o)
	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}
	if fn.arity != 0 {
		return nil, fmt.Errorf("expected function arity 0, got %d", fn.arity)
	}
	return *(*func() Object)(fn.nativePtr), nil
}

func (o Object) AsFunction1() (func(Object) Object, error) {
	if o.Kind() != InstanceKindFunction {
		return nil, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindFunction), kindToString(o.Kind()))
	}
	fn, err := getObjectValue[function](o)
	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}
	if fn.arity != 1 {
		return nil, fmt.Errorf("expected function arity 1, got %d", fn.arity)
	}
	return *(*func(Object) Object)(fn.nativePtr), nil
}

func (o Object) AsFunction2() (func(Object, Object) Object, error) {
	if o.Kind() != InstanceKindFunction {
		return nil, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindFunction), kindToString(o.Kind()))
	}
	fn, err := getObjectValue[function](o)
	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}
	if fn.arity != 2 {
		return nil, fmt.Errorf("expected function arity 2, got %d", fn.arity)
	}
	return *(*func(Object, Object) Object)(fn.nativePtr), nil
}

func (o Object) AsFunction3() (func(Object, Object, Object) Object, error) {
	if o.Kind() != InstanceKindFunction {
		return nil, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindFunction), kindToString(o.Kind()))
	}
	fn, err := getObjectValue[function](o)
	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}
	if fn.arity != 3 {
		return nil, fmt.Errorf("expected function arity 3, got %d", fn.arity)
	}
	return *(*func(Object, Object, Object) Object)(fn.nativePtr), nil
}

func (o Object) AsFunction4() (func(Object, Object, Object, Object) Object, error) {
	if o.Kind() != InstanceKindFunction {
		return nil, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindFunction), kindToString(o.Kind()))
	}
	fn, err := getObjectValue[function](o)
	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}
	if fn.arity != 4 {
		return nil, fmt.Errorf("expected function arity 4, got %d", fn.arity)
	}
	return *(*func(Object, Object, Object, Object) Object)(fn.nativePtr), nil
}

func (o Object) AsFunction5() (func(Object, Object, Object, Object, Object) Object, error) {
	if o.Kind() != InstanceKindFunction {
		return nil, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindFunction), kindToString(o.Kind()))
	}
	fn, err := getObjectValue[function](o)
	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}
	if fn.arity != 5 {
		return nil, fmt.Errorf("expected function arity 5, got %d", fn.arity)
	}
	return *(*func(Object, Object, Object, Object, Object) Object)(fn.nativePtr), nil
}

func (o Object) AsFunction6() (func(Object, Object, Object, Object, Object, Object) Object, error) {
	if o.Kind() != InstanceKindFunction {
		return nil, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindFunction), kindToString(o.Kind()))
	}
	fn, err := getObjectValue[function](o)
	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}
	if fn.arity != 6 {
		return nil, fmt.Errorf("expected function arity 6, got %d", fn.arity)
	}
	return *(*func(Object, Object, Object, Object, Object, Object) Object)(fn.nativePtr), nil
}

func (o Object) AsFunction7() (func(Object, Object, Object, Object, Object, Object, Object) Object, error) {
	if o.Kind() != InstanceKindFunction {
		return nil, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindFunction), kindToString(o.Kind()))
	}
	fn, err := getObjectValue[function](o)
	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}
	if fn.arity != 7 {
		return nil, fmt.Errorf("expected function arity 7, got %d", fn.arity)
	}
	return *(*func(Object, Object, Object, Object, Object, Object, Object) Object)(fn.nativePtr), nil
}

func (o Object) AsFunction8() (func(Object, Object, Object, Object, Object, Object, Object, Object) Object, error) {
	if o.Kind() != InstanceKindFunction {
		return nil, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindFunction), kindToString(o.Kind()))
	}
	fn, err := getObjectValue[function](o)
	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}
	if fn.arity != 8 {
		return nil, fmt.Errorf("expected function arity 8, got %d", fn.arity)
	}
	return *(*func(Object, Object, Object, Object, Object, Object, Object, Object) Object)(fn.nativePtr), nil
}

func (o Object) call(stack []Object) ([]Object, error) {
	if o.Kind() != InstanceKindFunction {
		return nil, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindFunction), kindToString(o.Kind()))
	}
	fn, err := getObjectValue[function](o)
	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}
	var result Object
	switch fn.arity {
	case 0:
		result = (*(*func() Object)(fn.nativePtr))()
	case 1:
		result = (*(*func(Object) Object)(fn.nativePtr))(stack[len(stack)-1])
	case 2:
		result = (*(*func(Object, Object) Object)(fn.nativePtr))(stack[len(stack)-2], stack[len(stack)-1])
	case 3:
		result = (*(*func(Object, Object, Object) Object)(fn.nativePtr))(stack[len(stack)-3], stack[len(stack)-2], stack[len(stack)-1])
	case 4:
		result = (*(*func(Object, Object, Object, Object) Object)(fn.nativePtr))(stack[len(stack)-4], stack[len(stack)-3], stack[len(stack)-2], stack[len(stack)-1])
	case 5:
		result = (*(*func(Object, Object, Object, Object, Object) Object)(fn.nativePtr))(stack[len(stack)-5], stack[len(stack)-4], stack[len(stack)-3], stack[len(stack)-2], stack[len(stack)-1])
	case 6:
		result = (*(*func(Object, Object, Object, Object, Object, Object) Object)(fn.nativePtr))(stack[len(stack)-6], stack[len(stack)-5], stack[len(stack)-4], stack[len(stack)-3], stack[len(stack)-2], stack[len(stack)-1])
	case 7:
		result = (*(*func(Object, Object, Object, Object, Object, Object, Object) Object)(fn.nativePtr))(stack[len(stack)-7], stack[len(stack)-6], stack[len(stack)-5], stack[len(stack)-4], stack[len(stack)-3], stack[len(stack)-2], stack[len(stack)-1])
	case 8:
		result = (*(*func(Object, Object, Object, Object, Object, Object, Object, Object) Object)(fn.nativePtr))(stack[len(stack)-8], stack[len(stack)-7], stack[len(stack)-6], stack[len(stack)-5], stack[len(stack)-4], stack[len(stack)-3], stack[len(stack)-2], stack[len(stack)-1])
	default:
		return nil, fmt.Errorf("unsupported function arity: %d", fn.arity)
	}
	return append(stack[:len(stack)-int(fn.arity)], result), nil
}

func (o Object) asClosure() (closure, error) {
	if o.Kind() != InstanceKindClosure {
		return closure{}, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindClosure), kindToString(o.Kind()))
	}
	return getObjectValue[closure](o)
}

func (o Object) asPattern() (pattern, error) {
	if o.Kind() != instanceKindPattern {
		return pattern{}, fmt.Errorf("expected %s, got %s", kindToString(instanceKindPattern), kindToString(o.Kind()))
	}
	return getObjectValue[pattern](o)
}

func (o Object) ConstEqualsTo(x Object) (bool, error) {
	if o.Kind() != x.Kind() {
		return false, fmt.Errorf("types are not equal %s vs %s", kindToString(o.Kind()), kindToString(x.Kind()))
	}
	switch o.Kind() {
	case InstanceKindUnit:
		return true, nil
	case InstanceKindChar:
		return getObjectValue[rune](o) == getObjectValue[rune](x), nil
	case InstanceKindInt:
		return getObjectValue[int](o) == getObjectValue[int](x), nil
	case InstanceKindFloat:
		return getObjectValue[float64](o) == getObjectValue[float64](x), nil
	case InstanceKindString:
		return getObjectValue[string](o) == getObjectValue[string](x), nil
	default:
		return false, fmt.Errorf("expected const comparison %s vs %s", kindToString(o.Kind()), kindToString(x.Kind()))
	}
}

func getObjectValue[T any](o Object) (T, error) {
	av, err := o.arena.at(o.index)
	if err != nil {
		var t T
		return t, fmt.Errorf("failed to get object value: %w", err)
	}
	v, ok := av.(T)
	if !ok {
		var t T
		return t, fmt.Errorf("failed to cast object value to %T", t)
	}
	return v, nil
}

type recordField struct {
	key    Object
	value  Object
	parent int
}

type tupleItem struct {
	value Object
	next  int
}

type listItem struct {
	value Object
	next  int
}

type option struct {
	fullName Object
	values   Object
}

type function struct {
	nativePtr unsafe.Pointer
	arity     uint8
}

type closure struct {
	fn      bytecode.Func
	curried Object
}

type pattern struct {
	name  Object //TODO: can hold only index, its always string
	items Object //TODO: can hold only index, its always list
	kind  bytecode.PatternKind
}

type InstanceKind uint8

const (
	InstanceKindUnit InstanceKind = iota
	InstanceKindChar
	InstanceKindInt
	InstanceKindFloat
	InstanceKindString
	InstanceKindRecord
	InstanceKindTuple
	InstanceKindList
	InstanceKindOption
	InstanceKindFunction
	InstanceKindClosure
	InstanceKindNative
	instanceKindPattern
	InstanceKindUnknown
)

func kindToString(kind InstanceKind) string {
	switch kind {
	case InstanceKindUnit:
		return "Unit"
	case InstanceKindChar:
		return "Char"
	case InstanceKindInt:
		return "Int"
	case InstanceKindFloat:
		return "Float"
	case InstanceKindString:
		return "String"
	case InstanceKindRecord:
		return "Record"
	case InstanceKindTuple:
		return "Tuple"
	case InstanceKindList:
		return "List"
	case InstanceKindOption:
		return "Option"
	case InstanceKindFunction:
		return "Function"
	case InstanceKindClosure:
		return "Closure"
	case InstanceKindNative:
		return "Native"
	case instanceKindPattern:
		return "Pattern"
	default:
		return "Unknown"
	}
}

package runtime

import (
	"fmt"
	"unsafe"
)

func ToUnit(rt *Runtime, o Object) (struct{}, error) {
	if o.Kind() != InstanceKindUnit {
		return struct{}{}, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindUnit), kindToString(o.Kind()))
	}
	return struct{}{}, nil
}

func ToChar(rt *Runtime, o Object) (TChar, error) {
	if o.Kind() != InstanceKindChar {
		return 0, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindChar), kindToString(o.Kind()))
	}
	return getObjectValue[TChar](rt, o)
}

func ToInt(rt *Runtime, o Object) (TInt, error) {
	if o.Kind() != InstanceKindInt {
		return 0, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindInt), kindToString(o.Kind()))
	}
	return getObjectValue[TInt](rt, o)
}

func ToFloat(rt *Runtime, o Object) (TFloat, error) {
	if o.Kind() != InstanceKindFloat {
		return 0, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindFloat), kindToString(o.Kind()))
	}
	return getObjectValue[TFloat](rt, o)
}

func ToString(rt *Runtime, o Object) (TString, error) {
	if o.Kind() != InstanceKindString {
		return "", fmt.Errorf("expected %s, got %s", kindToString(InstanceKindString), kindToString(o.Kind()))
	}
	return getObjectValue[TString](rt, o)
}

func FindRecordField(rt *Runtime, o Object, name TString) (Object, bool, error) {
	if o.Kind() != InstanceKindRecord {
		return invalidObject, false, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindRecord), kindToString(o.Kind()))
	}
	it := o
	for it.valid() {
		f, err := getObjectValue[recordField](rt, it)
		if err != nil {
			return invalidObject, false, fmt.Errorf("failed to get record field: %w", err)
		}
		key, err := ToString(rt, f.key)
		if err != nil {
			return invalidObject, false, fmt.Errorf("failed to unwrap key: %w", err)
		}
		if key == name {
			return f.value, true, nil
		}
		it = f.parent
	}
	return invalidObject, false, nil
}

func UpdateRecordField(rt *Runtime, o Object, key Object, value Object) (Object, error) {
	if o.Kind() != InstanceKindRecord {
		return invalidObject, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindRecord), kindToString(o.Kind()))
	}
	if key.Kind() != InstanceKindString {
		return invalidObject, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindString), kindToString(key.Kind()))
	}
	return addObject(rt, InstanceKindRecord, recordField{key: key, value: value, parent: o}), nil
}

func ToRecordFields(rt *Runtime, o Object) ([]struct {
	k TString
	v Object
}, error) {
	if o.Kind() != InstanceKindRecord {
		return nil, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindRecord), kindToString(o.Kind()))
	}
	var r []struct {
		k TString
		v Object
	}
	it := o
	for it.valid() {
		f, err := getObjectValue[recordField](rt, it)
		if err != nil {
			return nil, fmt.Errorf("failed to get record field: %w", err)
		}
		k, err := ToString(rt, f.key)
		if err != nil {
			return nil, fmt.Errorf("failed to unwrap key: %w", err)
		}
		r = append(r, struct {
			k TString
			v Object
		}{k: k, v: f.value})
		it = f.parent
	}
	return r, nil
}

func ToTuple(rt *Runtime, o Object) ([]Object, error) {
	if o.Kind() != InstanceKindTuple {
		return nil, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindTuple), kindToString(o.Kind()))
	}
	var r []Object
	it := o
	for it.valid() {
		f, err := getObjectValue[tupleItem](rt, it)
		if err != nil {
			return nil, fmt.Errorf("failed to get tuple item: %w", err)
		}
		r = append(r, f.value)
		it = f.next
	}
	return r, nil

}

func toListItem(rt *Runtime, o Object) (listItem, error) {
	if o.Kind() != InstanceKindList {
		return listItem{}, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindList), kindToString(o.Kind()))
	}
	return getObjectValue[listItem](rt, o)
}

func ToList(rt *Runtime, o Object) ([]Object, error) {
	if o.Kind() != InstanceKindList {
		return nil, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindList), kindToString(o.Kind()))
	}
	var r []Object
	it := o
	for it.valid() {
		f, err := getObjectValue[listItem](rt, it)
		if err != nil {
			return nil, fmt.Errorf("failed to get list item: %w", err)
		}
		r = append(r, f.value)
		it = f.next
	}
	return r, nil
}

func ToOption(rt *Runtime, o Object) (Object, []Object, error) {
	if o.Kind() != InstanceKindOption {
		return invalidObject, nil, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindOption), kindToString(o.Kind()))
	}
	opt, err := getObjectValue[option](rt, o)
	if err != nil {
		return invalidObject, nil, fmt.Errorf("failed to get option: %w", err)
	}
	values, err := ToList(rt, opt.values)
	if err != nil {
		return invalidObject, nil, fmt.Errorf("failed to unwrap option values: %w", err)
	}
	return opt.fullName, values, nil
}

func ToBool(rt *Runtime, o Object) (bool, error) {
	if o.Kind() != InstanceKindOption {
		return false, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindOption), kindToString(o.Kind()))
	}
	if o.index() == 1 {
		return true, nil

	}
	if o.index() == 0 {
		return false, nil
	}
	optName, _, err := ToOption(rt, o)
	if err != nil {
		return false, fmt.Errorf("failed to unwrap option: %w", err)
	}
	name, err := ToString(rt, optName)
	if err != nil {
		return false, fmt.Errorf("failed to unwrap option name: %w", err)
	}
	if name == kTrue {
		return true, nil
	}
	if name == kFalse {
		return false, nil
	}
	return false, fmt.Errorf("expected Boolean type")
}

func callFunc(ptr unsafe.Pointer, args []Object) (Object, error) {
	var result Object
	switch len(args) {
	case 0:
		result = (*(*func() Object)(ptr))()
	case 1:
		result = (*(*func(Object) Object)(ptr))(args[0])
	case 2:
		result = (*(*func(Object, Object) Object)(ptr))(args[0], args[1])
	case 3:
		result = (*(*func(Object, Object, Object) Object)(ptr))(args[0], args[1], args[2])
	case 4:
		result = (*(*func(Object, Object, Object, Object) Object)(ptr))(args[0], args[1], args[2], args[3])
	case 5:
		result = (*(*func(Object, Object, Object, Object, Object) Object)(ptr))(args[0], args[1], args[2], args[3], args[4])
	case 6:
		result = (*(*func(Object, Object, Object, Object, Object, Object) Object)(ptr))(args[0], args[1], args[2], args[3], args[4], args[5])
	case 7:
		result = (*(*func(Object, Object, Object, Object, Object, Object, Object) Object)(ptr))(args[0], args[1], args[2], args[3], args[4], args[5], args[6])
	case 8:
		result = (*(*func(Object, Object, Object, Object, Object, Object, Object, Object) Object)(ptr))(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7])
	default:
		return invalidObject, fmt.Errorf("unsupported function arity: %d", len(args))
	}
	return result, nil
}

func toClosure(rt *Runtime, o Object) (closure, error) {
	if o.Kind() != InstanceKindClosure {
		return closure{}, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindClosure), kindToString(o.Kind()))
	}
	return getObjectValue[closure](rt, o)
}

func toPattern(rt *Runtime, o Object) (pattern, error) {
	if o.Kind() != instanceKindPattern {
		return pattern{}, fmt.Errorf("expected %s, got %s", kindToString(instanceKindPattern), kindToString(o.Kind()))
	}
	return getObjectValue[pattern](rt, o)
}

func ToFunc(rt *Runtime, o Object) (unsafe.Pointer, uint8, error) {
	if o.Kind() != InstanceKindFunction {
		return nil, 0, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindFunction), kindToString(o.Kind()))
	}
	fn, err := getObjectValue[function](rt, o)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get function: %w", err)
	}
	return fn.ptr, fn.arity, nil
}

func ToNative(rt *Runtime, o Object) (unsafe.Pointer, unsafe.Pointer, error) {
	if o.Kind() != InstanceKindNative {
		return nil, nil, fmt.Errorf("expected %s, got %s", kindToString(InstanceKindNative), kindToString(o.Kind()))
	}
	n, err := getObjectValue[native](rt, o)
	return n.ptr, n.cmp, err
}

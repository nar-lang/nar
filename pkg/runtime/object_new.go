package runtime

import (
	"fmt"
	"nar-compiler/pkg/bytecode"
	"unsafe"
)

func (rt *Runtime) NewUnit() Object {
	return unitObject
}

func (rt *Runtime) NewChar(r TChar) Object {
	return rt.arenas[InstanceKindChar].(*typedArenaImpl[TChar]).add(r) //TODO: can be hashed to avoid duplication
}

func (rt *Runtime) NewInt(i TInt) Object {
	return rt.arenas[InstanceKindInt].(*typedArenaImpl[TInt]).add(i)
}

func (rt *Runtime) NewFloat(f TFloat) Object {
	return rt.arenas[InstanceKindFloat].(*typedArenaImpl[TFloat]).add(f)
}

func (rt *Runtime) NewString(s TString) Object {
	if s == "" {
		return newObject(InstanceKindString, 0)
	}
	return rt.arenas[InstanceKindString].(*typedArenaImpl[TString]).add(s) //TODO: can be hashed to avoid duplication
}

func (rt *Runtime) newObjectRecordRaw(valuesAndNames ...Object) Object {
	prev := newObject(InstanceKindRecord, invalidIndex)
	n := len(valuesAndNames)
	for i := 1; i < n; i += 2 {
		prev = rt.newRecordFiled(valuesAndNames[i], valuesAndNames[i-1], prev)
	}
	return prev
}

func (rt *Runtime) NewRecord(keys []TString, values []Object) Object {
	prev := newObject(InstanceKindRecord, invalidIndex)
	if len(keys) != len(values) {
		panic("keys and values must have the same length")
	}

	for i, key := range keys {
		prev = rt.newRecordFiled(rt.NewString(key), values[i], prev)
	}
	return prev
}

func (rt *Runtime) newRecordFiled(key Object, value Object, parent Object) Object {
	return rt.arenas[InstanceKindRecord].(*typedArenaImpl[recordField]).add(
		recordField{key: key, value: value, parent: parent})
}

func (rt *Runtime) NewTuple(items ...Object) Object {
	first := newObject(InstanceKindTuple, invalidIndex)
	for i := len(items) - 1; i >= 0; i-- {
		first = rt.newTupleItem(items[i], first)
	}
	return first
}

func (rt *Runtime) newTupleItem(value Object, next Object) Object {
	return rt.arenas[InstanceKindTuple].(*typedArenaImpl[tupleItem]).add(tupleItem{value: value, next: next})
}

func (rt *Runtime) NewList(elems ...Object) Object {
	first := newObject(InstanceKindList, invalidIndex)
	for i := len(elems) - 1; i >= 0; i-- {
		first = rt.newListItem(elems[i], first)
	}
	return first
}

func (rt *Runtime) newListItem(value Object, next Object) Object {
	return rt.arenas[InstanceKindList].(*typedArenaImpl[listItem]).add(listItem{value: value, next: next})
}

func (rt *Runtime) NewOptionWithTypeName(dataTypeName TString, optionName TString, values ...Object) Object {
	return rt.NewOption(dataTypeName+"#"+optionName, values...)
}

func (rt *Runtime) NewOption(optionName TString, values ...Object) Object {
	return rt.arenas[InstanceKindOption].(*typedArenaImpl[option]).add(
		option{fullName: rt.NewString(optionName), values: rt.NewList(values...)})
}

func (rt *Runtime) NewBool(b bool) Object {
	if b {
		return newObject(InstanceKindOption, 1)
	}
	return newObject(InstanceKindOption, 0)
}

func (rt *Runtime) NewFunc0(f func() Object) Object {
	return rt.newFunc(unsafe.Pointer(&f), 0)
}

func (rt *Runtime) NewFunc1(f func(Object) Object) Object {
	return rt.newFunc(unsafe.Pointer(&f), 1)
}

func (rt *Runtime) NewFunc2(f func(Object, Object) Object) Object {
	return rt.newFunc(unsafe.Pointer(&f), 2)
}

func (rt *Runtime) NewFunc3(f func(Object, Object, Object) Object) Object {
	return rt.newFunc(unsafe.Pointer(&f), 3)
}

func (rt *Runtime) NewFunc4(f func(Object, Object, Object, Object) Object) Object {
	return rt.newFunc(unsafe.Pointer(&f), 4)
}

func (rt *Runtime) NewFunc5(f func(Object, Object, Object, Object, Object) Object) Object {
	return rt.newFunc(unsafe.Pointer(&f), 5)
}

func (rt *Runtime) NewFunc6(f func(Object, Object, Object, Object, Object, Object) Object) Object {
	return rt.newFunc(unsafe.Pointer(&f), 6)
}

func (rt *Runtime) NewFunc7(f func(Object, Object, Object, Object, Object, Object, Object) Object) Object {
	return rt.newFunc(unsafe.Pointer(&f), 7)
}

func (rt *Runtime) NewFunc8(f func(Object, Object, Object, Object, Object, Object, Object, Object) Object) Object {
	return rt.newFunc(unsafe.Pointer(&f), 8)
}

func (rt *Runtime) newFunc(ptr unsafe.Pointer, arity uint8) Object {
	return rt.arenas[InstanceKindFunction].(*typedArenaImpl[function]).add(function{ptr: ptr, arity: arity})
}

func (rt *Runtime) newClosure(f bytecode.Func, curried ...Object) Object {
	return rt.arenas[InstanceKindClosure].(*typedArenaImpl[closure]).add(
		closure{fn: f, curried: rt.NewList(curried...)})
}

func (rt *Runtime) NewNative(ptr unsafe.Pointer, cmp unsafe.Pointer) Object {
	return rt.arenas[InstanceKindNative].(*typedArenaImpl[native]).add(
		native{ptr: ptr, cmp: cmp})
}

func (rt *Runtime) Clean(keepCapacity bool) {
	for _, arena := range rt.arenas {
		if arena != nil {
			arena.clean(keepCapacity)
		}
	}

	if keepCapacity {
		rt.callStack = rt.callStack[:0]
		rt.locals = rt.locals[:0]
		for k := range rt.cachedExpressions {
			delete(rt.cachedExpressions, k)
		}
	} else {
		rt.callStack = nil
		rt.locals = nil
		rt.cachedExpressions = map[bytecode.Pointer]Object{}
	}

	rt.arenas[InstanceKindString].(*typedArenaImpl[TString]).add("")
	_ = rt.NewOption(kFalse)
	_ = rt.NewOption(kTrue)
}

func (rt *Runtime) newPattern(name Object, items []Object, kind bytecode.PatternKind) (Object, error) {
	if kDebug {
		for _, item := range items {
			switch item.Kind() {
			case InstanceKindUnit:
			case InstanceKindChar:
			case InstanceKindInt:
			case InstanceKindFloat:
			case InstanceKindString:
			case instanceKindPattern:
				break
			default:
				return invalidObject, fmt.Errorf("pattern can only contain pattern items")
			}
		}
	}
	return rt.arenas[instanceKindPattern].(*typedArenaImpl[pattern]).add(
		pattern{name: name, items: rt.NewList(items...), kind: kind}), nil
}

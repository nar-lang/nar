package runtime

import (
	"fmt"
	"nar-compiler/pkg/bytecode"
	"unsafe"
)

type Arena struct {
	arenas            []typedArena
	objectStack       []Object
	patternStack      []Object
	callStack         []bytecode.StringHash
	locals            []local
	cachedExpressions map[bytecode.Pointer]Object
}

type local struct {
	name  string
	value Object
}

type typedArena interface {
	kind() InstanceKind
	add(x any) Object
	at(index int) (any, error)
	clean(keepCapacity bool)
}

type typedArenaImpl[T any] struct {
	kind_ InstanceKind
	items []T
}

func (a *typedArenaImpl[T]) at(index int) (any, error) {
	if index < 0 || index >= len(a.items) {
		return nil, fmt.Errorf("invalid object")
	}
	return a.items[index], nil
}

func (a *typedArenaImpl[T]) clean(keepCapacity bool) {
	if keepCapacity {
		a.items = a.items[:0]
	} else {
		a.items = nil
	}
}

func (a *typedArenaImpl[T]) add(x any) Object {
	sz := len(a.items)
	a.items = append(a.items, x.(T))
	return Object{arena: a, index: sz}
}

func (a *typedArenaImpl[T]) kind() InstanceKind {
	return a.kind_
}

func newTypedArenaWithKind(kind InstanceKind, initialCapacity int) typedArena {
	switch kind {
	case InstanceKindUnit:
		return nil
	case InstanceKindChar:
		return newTypedArena[rune](kind, initialCapacity)
	case InstanceKindInt:
		return newTypedArena[int64](kind, initialCapacity)
	case InstanceKindFloat:
		return newTypedArena[float64](kind, initialCapacity)
	case InstanceKindString:
		return newTypedArena[string](kind, initialCapacity)
	case InstanceKindRecord:
		return newTypedArena[recordField](kind, initialCapacity)
	case InstanceKindTuple:
		return newTypedArena[tupleItem](kind, initialCapacity)
	case InstanceKindList:
		return newTypedArena[listItem](kind, initialCapacity)
	case InstanceKindOption:
		return newTypedArena[option](kind, initialCapacity)
	case InstanceKindFunction:
		return newTypedArena[function](kind, initialCapacity)
	case InstanceKindClosure:
		return newTypedArena[closure](kind, initialCapacity)
	case InstanceKindNative:
		return newTypedArena[unsafe.Pointer](kind, initialCapacity)
	default:
		panic(fmt.Sprintf("unsupported kind: %v", kind))
	}
}

func newTypedArena[T any](kind InstanceKind, initialCapacity int) typedArena {
	return &typedArenaImpl[T]{kind_: kind, items: make([]T, 0, initialCapacity)}
}

func newArena(initialCapacity int) *Arena {
	a := &Arena{
		arenas:            make([]typedArena, InstanceKindUnknown),
		cachedExpressions: map[bytecode.Pointer]Object{},
	}
	for i := range a.arenas {
		a.arenas[i] = newTypedArenaWithKind(InstanceKind(i), initialCapacity)
	}
	a.Clean(true)
	return a
}

func (a *Arena) New(x any) (Object, error) {
	switch x := x.(type) {
	case Object:
		return x, nil
	case nil:
		return a.NewUnit(), nil
	case rune:
		return a.NewChar(x), nil
	case int64:
		return a.NewInt(x), nil
	case float64:
		return a.NewFloat(x), nil
	case string:
		return a.NewString(x), nil
	case map[any]any:
		return a.NewRecord(x)
	case []any:
		return a.NewList(x)
	case unsafe.Pointer:
		return a.NewNative(x), nil
	default:
		return invalidObject, fmt.Errorf("unsupported type: %T", x)
	}
}

func (a *Arena) NewUnit() Object {
	return unitObject
}

func (a *Arena) NewChar(r rune) Object {
	return a.arenas[InstanceKindChar].add(r) //TODO: can be hashed to avoid duplication
}

func (a *Arena) NewInt(i int64) Object {
	return a.arenas[InstanceKindInt].add(i)
}

func (a *Arena) NewFloat(f float64) Object {
	return a.arenas[InstanceKindFloat].add(f)
}

func (a *Arena) NewString(s string) Object {
	if s == "" {
		return Object{arena: a.arenas[InstanceKindString]}
	}
	return a.arenas[InstanceKindString].add(s) //TODO: can be hashed to avoid duplication
}

func (a *Arena) NewRecord(r map[any]any) (Object, error) {
	prev := Object{index: -1, arena: a.arenas[InstanceKindRecord]}
	for k, v := range r {
		key, err := a.New(k)
		if err != nil {
			return invalidObject, err
		}
		value, err := a.New(v)
		if err != nil {
			return invalidObject, err
		}
		prev = a.newRecordFiled(key, value, prev.index)
	}
	return prev, nil
}

func (a *Arena) newObjectRecord(valuesAndNames ...Object) Object {
	prev := Object{index: -1, arena: a.arenas[InstanceKindRecord]}
	n := len(valuesAndNames)
	for i := 1; i < n; i += 2 {
		prev = a.newRecordFiled(valuesAndNames[i], valuesAndNames[i-1], prev.index)
	}
	return prev
}

func (a *Arena) newRecordFiled(key Object, value Object, parent int) Object {
	return a.arenas[InstanceKindRecord].add(recordField{key: key, value: value, parent: parent})
}

func (a *Arena) NewObjectTuple(items ...Object) Object {
	first := Object{index: -1, arena: a.arenas[InstanceKindTuple]}
	for i := len(items) - 1; i >= 0; i-- {
		first = a.newTupleItem(items[i], first.index)
	}
	return first
}

func (a *Arena) NewTuple(elems []any) (Object, error) {
	first := Object{index: -1, arena: a.arenas[InstanceKindTuple]}
	for i := len(elems) - 1; i >= 0; i-- {
		item, err := a.New(elems[i])
		if err != nil {
			return invalidObject, err
		}
		first = a.newTupleItem(item, first.index)
	}
	return first, nil
}

func (a *Arena) newTupleItem(value Object, next int) Object {
	return a.arenas[InstanceKindTuple].add(tupleItem{value: value, next: next})
}

func (a *Arena) NewObjectList(elems ...Object) Object {
	first := Object{index: -1, arena: a.arenas[InstanceKindList]}
	for i := len(elems) - 1; i >= 0; i-- {
		first = a.newListItem(elems[i], first.index)
	}
	return first
}

func (a *Arena) NewList(elems []any) (Object, error) {
	first := Object{index: -1, arena: a.arenas[InstanceKindList]}
	for i := len(elems) - 1; i >= 0; i-- {
		item, err := a.New(elems[i])
		if err != nil {
			return invalidObject, err
		}
		first = a.newListItem(item, first.index)
	}
	return first, nil
}

func NewList[T any](a *Arena, list []T) (Object, error) {
	first := Object{index: -1, arena: a.arenas[InstanceKindList]}
	for i := len(list) - 1; i >= 0; i-- {
		item, err := a.New(list[i])
		if err != nil {
			return invalidObject, err
		}
		first = a.newListItem(item, first.index)
	}
	return first, nil
}

func (a *Arena) newListItem(value Object, next int) Object {
	return a.arenas[InstanceKindList].add(listItem{value: value, next: next})
}

func (a *Arena) NewOption(dataTypeName string, optionName string, args ...any) (Object, error) {
	return a.NewOptionWithFullName(dataTypeName+"#"+optionName, args...)
}

func (a *Arena) NewOptionWithFullName(optionName string, args ...any) (Object, error) {
	tt, err := a.NewTuple(args)
	if err != nil {
		return invalidObject, err
	}
	to := option{fullName: a.NewString(optionName), values: tt}
	return a.arenas[InstanceKindOption].add(to), nil
}

func (a *Arena) NewObjectOption(optionName string, values ...Object) Object {
	return a.arenas[InstanceKindOption].add(option{fullName: a.NewString(optionName), values: a.NewObjectList(values...)})
}

func (a *Arena) NewBool(b bool) Object {
	if b {
		return Object{arena: a.arenas[InstanceKindOption], index: 1}
	}
	return Object{arena: a.arenas[InstanceKindOption], index: 0}
}

func (a *Arena) NewFunc0(f func() Object) Object {
	return a.newFunc(unsafe.Pointer(&f), 0)
}

func (a *Arena) NewFunc1(f func(Object) Object) Object {
	return a.newFunc(unsafe.Pointer(&f), 1)
}

func (a *Arena) NewFunc2(f func(Object, Object) Object) Object {
	return a.newFunc(unsafe.Pointer(&f), 2)
}

func (a *Arena) NewFunc3(f func(Object, Object, Object) Object) Object {
	return a.newFunc(unsafe.Pointer(&f), 3)
}

func (a *Arena) NewFunc4(f func(Object, Object, Object, Object) Object) Object {
	return a.newFunc(unsafe.Pointer(&f), 4)
}

func (a *Arena) NewFunc5(f func(Object, Object, Object, Object, Object) Object) Object {
	return a.newFunc(unsafe.Pointer(&f), 5)
}

func (a *Arena) NewFunc6(f func(Object, Object, Object, Object, Object, Object) Object) Object {
	return a.newFunc(unsafe.Pointer(&f), 6)
}

func (a *Arena) NewFunc7(f func(Object, Object, Object, Object, Object, Object, Object) Object) Object {
	return a.newFunc(unsafe.Pointer(&f), 7)
}

func (a *Arena) NewFunc8(f func(Object, Object, Object, Object, Object, Object, Object, Object) Object) Object {
	return a.newFunc(unsafe.Pointer(&f), 8)
}

func (a *Arena) newFunc(ptr unsafe.Pointer, arity uint8) Object {
	return a.arenas[InstanceKindFunction].add(function{nativePtr: ptr, arity: arity})
}

func (a *Arena) newClosure(f bytecode.Func, curried ...Object) Object {
	return a.arenas[InstanceKindClosure].add(closure{fn: f, curried: a.NewObjectList(curried...)})
}

func (a *Arena) NewNative(ptr unsafe.Pointer) Object {
	return a.arenas[InstanceKindNative].add(ptr)
}

func (a *Arena) Clean(keepCapacity bool) {
	for _, arena := range a.arenas {
		arena.clean(keepCapacity)
	}

	if keepCapacity {
		a.callStack = a.callStack[:0]
		a.locals = a.locals[:0]
		for k := range a.cachedExpressions {
			delete(a.cachedExpressions, k)
		}
		a.objectStack = a.objectStack[:0]
		a.patternStack = a.patternStack[:0]
	} else {
		a.callStack = nil
		a.locals = nil
		a.cachedExpressions = map[bytecode.Pointer]Object{}
		a.objectStack = nil
		a.patternStack = nil
	}

	_, _ = a.NewOptionWithFullName(kFalse)
	_, _ = a.NewOptionWithFullName(kTrue)
	a.arenas[InstanceKindString].add("")
}

func (a *Arena) newPattern(name Object, items []Object) (Object, error) {
	if kDebug {
		for _, item := range items {
			switch item.arena.kind() {
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
	return a.arenas[instanceKindPattern].add(pattern{name: name, items: a.NewObjectList(items...)}), nil
}

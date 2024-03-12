package runtime

import (
	"fmt"
)

type typedArena interface {
	kind() InstanceKind
	clean(emptyMemory bool)
}

type typedArenaImpl[T any] struct {
	kind_ InstanceKind
	items []T
}

func (a *typedArenaImpl[T]) at(index uint64) (any, error) {
	if index >= uint64(len(a.items)) {
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

func (a *typedArenaImpl[T]) add(x T) Object {
	sz := len(a.items)
	a.items = append(a.items, x)
	return newObject(a.kind_, uint64(sz))
}

func (a *typedArenaImpl[T]) kind() InstanceKind {
	return a.kind_
}

func newTypedArenaWithKind(kind InstanceKind, initialCapacity int) typedArena {
	switch kind {
	case InstanceKindUnknown:
		return nil
	case InstanceKindUnit:
		return nil
	case InstanceKindChar:
		return newTypedArena[TChar](kind, initialCapacity)
	case InstanceKindInt:
		return newTypedArena[TInt](kind, initialCapacity)
	case InstanceKindFloat:
		return newTypedArena[TFloat](kind, initialCapacity)
	case InstanceKindString:
		return newTypedArena[TString](kind, initialCapacity)
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
		return newTypedArena[native](kind, initialCapacity)
	case instanceKindPattern:
		return newTypedArena[pattern](kind, initialCapacity)
	default:
		panic(fmt.Sprintf("unsupported kind: %v", kind))
	}
}

func newTypedArena[T any](kind InstanceKind, initialCapacity int) typedArena {
	return &typedArenaImpl[T]{kind_: kind, items: make([]T, 0, initialCapacity)}
}

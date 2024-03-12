package runtime

import (
	"fmt"
	"nar-compiler/pkg/bytecode"
	"unsafe"
)

const kTrue = "Nar.Base.Basics.Bool#True"
const kFalse = "Nar.Base.Basics.Bool#False"

var invalidIndex = uint64(0x0010000000000000)
var invalidObject = newObject(InstanceKindUnknown, 0)
var unitObject = newObject(InstanceKindUnit, 0)

type Object uint64

func newObject(kind InstanceKind, index uint64) Object {
	return Object((uint64(kind) << 56) | (index & 0x00ffffffffffffff))
}

func (o Object) Kind() InstanceKind {
	return InstanceKind((o >> 56) & 0xff)
}

func (o Object) index() uint64 {
	return uint64(o) & 0x00ffffffffffffff
}

func (o Object) valid() bool {
	return (uint64(o) & invalidIndex) == 0
}

func (o Object) invalid() bool {
	return (uint64(o) & invalidIndex) != 0
}

type InstanceKind uint8

const (
	InstanceKindUnknown InstanceKind = iota
	InstanceKindUnit
	InstanceKindInt
	InstanceKindFloat
	InstanceKindString
	InstanceKindChar
	InstanceKindRecord
	InstanceKindTuple
	InstanceKindList
	InstanceKindOption
	InstanceKindFunction
	InstanceKindClosure
	InstanceKindNative
	instanceKindPattern
	instanceKindCount
)

func getObjectValue[T any](rt *Runtime, o Object) (T, error) {
	arena := rt.arenas[o.Kind()]
	av, err := arena.(*typedArenaImpl[T]).at(o.index())
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

func addObject[T any](rt *Runtime, kind InstanceKind, value T) Object {
	arena := rt.arenas[kind]
	return arena.(*typedArenaImpl[T]).add(value)
}

func objEquals[T comparable](ox, oy Object, rt *Runtime) (bool, error) {
	l, err := getObjectValue[T](rt, ox)
	if err != nil {
		return false, err
	}
	r, err := getObjectValue[T](rt, oy)
	if err != nil {
		return false, err
	}
	return l == r, nil
}

func constEqualsTo(rt *Runtime, x, y Object) (bool, error) {
	if x.Kind() != y.Kind() {
		return false, fmt.Errorf("types are not equal %s vs %s", kindToString(x.Kind()), kindToString(y.Kind()))
	}

	switch x.Kind() {
	case InstanceKindUnit:
		return true, nil
	case InstanceKindChar:
		return objEquals[TChar](x, y, rt)
	case InstanceKindInt:
		return objEquals[TInt](x, y, rt)
	case InstanceKindFloat:
		return objEquals[TFloat](x, y, rt)
	case InstanceKindString:
		return objEquals[TString](x, y, rt)
	default:
		return false, fmt.Errorf("expected const comparison %s vs %s",
			kindToString(x.Kind()), kindToString(y.Kind()))
	}
}

type TChar rune
type TInt int64
type TFloat float64
type TString string
type TSize uint64

type recordField struct {
	key    Object
	value  Object
	parent Object
}

type tupleItem struct {
	value Object
	next  Object
}

type listItem struct {
	value Object
	next  Object
}

type option struct {
	fullName Object
	values   Object
}

type function struct {
	ptr   unsafe.Pointer
	arity uint8
}

type closure struct {
	fn      bytecode.Func
	curried Object
}

type native struct {
	ptr unsafe.Pointer
	cmp unsafe.Pointer
}

type pattern struct {
	name  Object
	items Object
	kind  bytecode.PatternKind
}

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

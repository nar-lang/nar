package bytecode

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
)

type OpKind uint32
type StringHash uint32
type ConstHash uint32
type Pointer uint32
type PatternKind uint8
type ConstKind uint8
type StackKind uint8
type SwapPopMode uint8
type ObjectKind uint8

const (
	opKindNone OpKind = iota
	OpKindLoadLocal
	OpKindLoadGlobal
	OpKindLoadConst
	OpKindSwapPop
	OpKindApply
	OpKindCall
	OpKindMatch
	OpKindJump
	OpKindMakeObject
	OpKindMakePattern
	OpKindAccess
	OpKindUpdate
)
const (
	patternKindNone PatternKind = iota
	PatternKindAlias
	PatternKindAny
	PatternKindCons
	PatternKindConst
	PatternKindDataOption
	PatternKindList
	PatternKindNamed
	PatternKindRecord
	PatternKindTuple
)
const (
	constKindNone ConstKind = iota
	ConstKindUnit
	ConstKindChar
	ConstKindInt
	ConstKindFloat
	ConstKindString
)
const (
	stackKindNone StackKind = iota
	StackKindObject
	StackKindPattern
)
const (
	objectKindNone ObjectKind = iota
	ObjectKindList
	ObjectKindTuple
	ObjectKindRecord
	ObjectKindData
)
const (
	swapPopModeNone SwapPopMode = iota
	SwapPopModeBoth
	SwapPopModePop
)

type Op interface {
	Word() uint64
	WithDelta(delta int32) Op
}

// loadLocal adds named local object to the top of the stack
type loadLocal struct {
	Name StringHash
}

func (op loadLocal) WithDelta(delta int32) Op {
	panic("should not be called")
}

func (op loadLocal) Word() uint64 {
	return uint64(OpKindLoadLocal) |
		(uint64(op.Name) << 32)
}

func AppendLoadLocal(
	name string, loc ast.Location, ops []Op, locations []ast.Location, binary *Binary,
) ([]Op, []ast.Location) {
	return append(ops, loadLocal{Name: binary.HashString(name)}),
		append(locations, loc)
}

// loadGlobal adds global object to the top of the stack
type loadGlobal struct {
	Pointer Pointer
}

func (op loadGlobal) WithDelta(delta int32) Op {
	panic("should not be called")
}

func (op loadGlobal) Word() uint64 {
	return uint64(OpKindLoadGlobal) |
		(uint64(op.Pointer) << 32)
}

func AppendLoadGlobal(
	ptr Pointer, loc ast.Location, ops []Op, locations []ast.Location,
) ([]Op, []ast.Location) {
	return append(ops, loadGlobal{Pointer: ptr}),
		append(locations, loc)
}

// loadConst adds const value object to the top of the stack
type loadConst struct {
	Stack StackKind
	Kind  ConstKind
	Value ConstHash
}

func (op loadConst) WithDelta(delta int32) Op {
	panic("should not be called")
}

func (op loadConst) Word() uint64 {
	return uint64(OpKindLoadConst) |
		(uint64(op.Stack) << 8) |
		(uint64(op.Kind) << 16) |
		(uint64(op.Value) << 32)
}

func AppendLoadConstValue(
	c ast.ConstValue, stack StackKind, loc ast.Location,
	ops []Op, locations []ast.Location, binary *Binary,
) ([]Op, []ast.Location) {
	switch c.(type) {
	case ast.CUnit:
		{
			return append(ops, loadConst{
					Stack: stack,
					Kind:  ConstKindUnit,
					Value: 0,
				}),
				append(locations, loc)
		}
	case ast.CChar:
		{
			return append(ops, loadConst{
					Stack: stack,
					Kind:  ConstKindChar,
					Value: ConstHash(c.(ast.CChar).Value),
				}),
				append(locations, loc)
		}
	case ast.CInt:
		{
			return append(ops, loadConst{
					Stack: stack,
					Kind:  ConstKindInt,
					Value: binary.HashConst(PackedInt{Value: c.(ast.CInt).Value}),
				}),
				append(locations, loc)
		}
	case ast.CFloat:
		{
			return append(ops, loadConst{
					Stack: stack,
					Kind:  ConstKindFloat,
					Value: binary.HashConst(PackedFloat{Value: c.(ast.CFloat).Value}),
				}),
				append(locations, loc)
		}
	case ast.CString:
		{
			return append(ops, loadConst{
					Stack: stack,
					Kind:  ConstKindString,
					Value: ConstHash(binary.HashString(c.(ast.CString).Value)),
				}),
				append(locations, loc)
		}
	default:
		panic("impossible case")
	}
}

// apply executes the function from the top of the stack.
// Arguments are taken from the top of the stack in reverse order
// (topmost object is the last arg). Returned value is left on the top of the stack.
// In case of NumArgs is less than number of function parameters it creates
// a closure and leaves it on the top of the stack
type apply struct {
	NumArgs uint8
}

func (op apply) WithDelta(delta int32) Op {
	panic("should not be called")
}

func (op apply) Word() uint64 {
	return uint64(OpKindApply) |
		(uint64(op.NumArgs) << 8)
}

func AppendApply(numArgs int, loc ast.Location, ops []Op, locations []ast.Location,
) ([]Op, []ast.Location) {
	if numArgs > 255 {
		panic(common.NewErrorAt(loc, "function cannot be applied with more than 255 arguments").Error())
	}
	return append(ops, apply{NumArgs: uint8(numArgs)}),
		append(locations, loc)
}

// call executes native function.
// Arguments are taken from the top of the stack in reverse order
// (topmost object is last arg). Returned value is left on the top of the stack.
type call struct {
	Name    StringHash
	NumArgs uint8
}

func (op call) WithDelta(delta int32) Op {
	panic("should not be called")
}

func (op call) Word() uint64 {
	return uint64(OpKindCall) |
		(uint64(op.NumArgs) << 8) |
		(uint64(op.Name) << 32)
}

func AppendCall(name string, numArgs int, loc ast.Location, ops []Op, locations []ast.Location, binary *Binary,
) ([]Op, []ast.Location) {
	if numArgs > 255 {
		panic(common.NewErrorAt(loc, "function cannot be called with more than 255 arguments").Error())
	}
	return append(ops, call{Name: binary.HashString(name), NumArgs: uint8(numArgs)}),
		append(locations, loc)
}

// match tries to match pattern with object on the top of the stack.
// If it cannot be matched it moves on delta ops
// If it matches successfully - locals are extracted from pattern
// Matched object is left on the top of the stack in both cases
type match struct {
	JumpDelta int32
}

func (op match) WithDelta(delta int32) Op {
	op.JumpDelta = delta
	return op
}

func (op match) Word() uint64 {
	return uint64(OpKindMatch) |
		(uint64(op.JumpDelta) << 32)
}

func AppendMatch(jumpDelta int, loc ast.Location, ops []Op, locations []ast.Location) ([]Op, []ast.Location) {
	return append(ops, match{JumpDelta: int32(jumpDelta)}),
		append(locations, loc)
}

// jump moves on delta ops unconditional
type jump struct {
	Delta int32
}

func (op jump) WithDelta(delta int32) Op {
	op.Delta = delta
	return op
}

func (op jump) Word() uint64 {
	return uint64(OpKindJump) |
		(uint64(op.Delta) << 32)
}

func AppendJump(delta int, loc ast.Location,
	ops []Op, locations []ast.Location,
) ([]Op, []ast.Location) {
	return append(ops, jump{Delta: int32(delta)}),
		append(locations, loc)
}

// makeObject creates an object on stack.
// List items stored on stack in reverse order (topmost object is the last item)
// Record fields stored as repeating pairs const string and value (field name is on the top of the stack)
// Data stores option name as const string on the top of the stack and
// args after that in reverse order (topmost is the last arg)
type makeObject struct {
	Kind    ObjectKind
	NumArgs uint32
}

func (op makeObject) WithDelta(delta int32) Op {
	panic("should not be called")
}

func (op makeObject) Word() uint64 {
	return uint64(OpKindMakeObject) |
		(uint64(op.Kind) << 8) |
		(uint64(op.NumArgs) << 32)
}

func AppendMakeObject(kind ObjectKind, numArgs int, loc ast.Location,
	ops []Op, locations []ast.Location,
) ([]Op, []ast.Location) {
	return append(ops, makeObject{Kind: kind, NumArgs: uint32(numArgs)}),
		append(locations, loc)

}

// makePattern creates pattern object
// Arguments are taken from the top of the stack in reverse order
// (topmost object is the last arg). Created object is left on the top of the stack.
type makePattern struct {
	Kind      PatternKind
	Name      StringHash
	NumNested uint8
}

func (op makePattern) WithDelta(delta int32) Op {
	panic("should not be called")
}

func (op makePattern) Word() uint64 {
	return uint64(OpKindMakePattern) |
		(uint64(op.Kind) << 8) |
		(uint64(op.NumNested) << 16) |
		(uint64(op.Name) << 32)
}

func AppendMakePattern(
	kind PatternKind, name string, numNested int,
	loc ast.Location, ops []Op, locations []ast.Location, binary *Binary,
) ([]Op, []ast.Location) {
	if numNested > 255 {
		panic(common.NewErrorAt(loc, "pattern cannot contain more than 255 nested patterns").Error())
	}
	return append(ops, makePattern{Kind: kind, Name: binary.HashString(name), NumNested: uint8(numNested)}),
		append(locations, loc)
}

// access takes record object from the top of the stack and leaves its field on the stack
type access struct {
	Field StringHash
}

func (op access) WithDelta(delta int32) Op {
	panic("should not be called")
}

func (op access) Word() uint64 {
	return uint64(OpKindAccess) |
		(uint64(op.Field) << 32)
}

func AppendAccess(filed string, loc ast.Location, ops []Op, locations []ast.Location, binary *Binary,
) ([]Op, []ast.Location) {
	return append(ops, access{Field: binary.HashString(filed)}),
		append(locations, loc)
}

// update create new record with replaced field from the top of the stack and rest fields
// form the second record object from stack. Created record is left on the top of the stack
type update struct {
	Field StringHash
}

func (op update) WithDelta(delta int32) Op {
	panic("should not be called")
}

func (op update) Word() uint64 {
	return uint64(OpKindUpdate) |
		(uint64(op.Field) << 32)
}

func AppendUpdate(field string, loc ast.Location,
	ops []Op, locations []ast.Location, binary *Binary,
) ([]Op, []ast.Location) {
	return append(ops, update{Field: binary.HashString(field)}),
		append(locations, loc)
}

// swapPop removes second object from the top of the stack and leaves first object on the top of the stack
type swapPop struct {
	Mode SwapPopMode
}

func (op swapPop) WithDelta(delta int32) Op {
	panic("should not be called")
}

func (op swapPop) Word() uint64 {
	return uint64(OpKindSwapPop) |
		(uint64(op.Mode) << 8)
}

func AppendSwapPop(loc ast.Location, mode SwapPopMode,
	ops []Op, locations []ast.Location,
) ([]Op, []ast.Location) {
	return append(ops, swapPop{Mode: mode}),
		append(locations, loc)
}

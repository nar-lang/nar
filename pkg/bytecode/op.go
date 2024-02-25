package bytecode

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
	// OpKindLoadLocal adds named local object to the top of the stack
	OpKindLoadLocal
	// OpKindLoadGlobal adds global object to the top of the stack
	OpKindLoadGlobal
	// OpKindLoadConst adds const value object to the top of the stack
	OpKindLoadConst
	// OpKindApply executes the function from the top of the stack.
	// Arguments are taken from the top of the stack in reverse order
	// (topmost object is the last arg). Returned value is left on the top of the stack.
	// In case of NumArgs is less than number of function parameters it creates
	// a closure and leaves it on the top of the stack
	OpKindApply
	// OpKindCall executes native function.
	// Arguments are taken from the top of the stack in reverse order
	// (topmost object is last arg). Returned value is left on the top of the stack.
	OpKindCall
	// OpKindJump moves on delta ops unconditional
	// conditional jump tries to match pattern with object on the top of the stack.
	// If it cannot be matched it moves on delta ops
	// If it matches successfully - locals are extracted from pattern
	// Matched object is left on the top of the stack in both cases
	OpKindJump
	// OpKindMakeObject creates an object on stack.
	// List items stored on stack in reverse order (topmost object is the last item)
	// Record fields stored as repeating pairs const string and value (field name is on the top of the stack)
	// Data stores option name as const string on the top of the stack and
	// args after that in reverse order (topmost is the last arg)
	OpKindMakeObject
	// OpKindMakePattern creates pattern object
	// Arguments are taken from the top of the stack in reverse order
	// (topmost object is the last arg). Created object is left on the top of the stack.
	OpKindMakePattern
	// OpKindAccess takes record object from the top of the stack and leaves its field on the stack
	OpKindAccess
	// OpKindUpdate create new record with replaced field from the top of the stack and rest fields
	// form the second record object from stack. Created record is left on the top of the stack
	OpKindUpdate
	// OpKindSwapPop if pop mode - removes topmost object from the stack
	// if both mode - removes second object from the top of the stack
	OpKindSwapPop
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
	ObjectKindOption
)
const (
	swapPopModeNone SwapPopMode = iota
	SwapPopModeBoth
	SwapPopModePop
)

type Op uint64

func (o Op) Decompose() (OpKind, uint8, uint8, uint32) {
	return OpKind(o & 0xff), uint8((o >> 8) & 0xff), uint8((o >> 16) & 0xff), uint32(o >> 32)
}

func (o Op) WithDelta(i int32) Op {
	kind, b, c, _ := o.Decompose()
	return buildOp(kind, b, c, uint32(i))
}

func buildOp(kind OpKind, b uint8, c uint8, a uint32) Op {
	return Op(uint64(kind) | (uint64(b) << 8) | (uint64(c) << 16) | (uint64(a) << 32))
}

func AppendLoadLocal(
	name string, loc Location, ops []Op, locations []Location, binary *Binary, hash *BinaryHash,
) ([]Op, []Location) {
	return append(ops, buildOp(OpKindLoadLocal, 0, 0, uint32(hash.HashString(name, binary)))),
		append(locations, loc)
}

func AppendLoadGlobal(
	ptr Pointer, loc Location, ops []Op, locations []Location,
) ([]Op, []Location) {
	return append(ops, buildOp(OpKindLoadGlobal, 0, 0, uint32(ptr))),
		append(locations, loc)
}

func AppendLoadConstUnitValue(
	stack StackKind, loc Location,
	ops []Op, locations []Location, binary *Binary,
) ([]Op, []Location) {
	return append(ops, buildOp(OpKindLoadConst, uint8(stack), uint8(ConstKindUnit), 0)),
		append(locations, loc)
}

func AppendLoadConstCharValue(
	v rune, stack StackKind, loc Location,
	ops []Op, locations []Location, binary *Binary,
) ([]Op, []Location) {
	return append(ops, buildOp(OpKindLoadConst, uint8(stack), uint8(ConstKindChar), uint32(v))),
		append(locations, loc)
}

func AppendLoadConstIntValue(
	v int64, stack StackKind, loc Location,
	ops []Op, locations []Location, binary *Binary, hash *BinaryHash,
) ([]Op, []Location) {
	x := uint32(hash.HashConst(PackedInt{Value: v}, binary))
	return append(ops, buildOp(OpKindLoadConst, uint8(stack), uint8(ConstKindInt), x)),
		append(locations, loc)
}

func AppendLoadConstFloatValue(
	v float64, stack StackKind, loc Location,
	ops []Op, locations []Location, binary *Binary, hash *BinaryHash,
) ([]Op, []Location) {

	x := uint32(hash.HashConst(PackedFloat{Value: v}, binary))
	return append(ops, buildOp(OpKindLoadConst, uint8(stack), uint8(ConstKindFloat), x)),
		append(locations, loc)
}

func AppendLoadConstStringValue(
	v string, stack StackKind, loc Location,
	ops []Op, locations []Location, binary *Binary, hash *BinaryHash,
) ([]Op, []Location) {
	x := uint32(hash.HashString(v, binary))
	return append(ops, buildOp(OpKindLoadConst, uint8(stack), uint8(ConstKindString), x)),
		append(locations, loc)
}

func AppendApply(numArgs uint8, loc Location, ops []Op, locations []Location,
) ([]Op, []Location) {
	return append(ops, buildOp(OpKindApply, numArgs, 0, 0)),
		append(locations, loc)
}

func AppendCall(
	name string, numArgs uint8, loc Location,
	ops []Op, locations []Location, binary *Binary, hash *BinaryHash,
) ([]Op, []Location) {
	return append(ops, buildOp(OpKindCall, numArgs, 0, uint32(hash.HashString(name, binary)))),
		append(locations, loc)
}

func AppendJump(jumpDelta int, conditional bool, loc Location, ops []Op, locations []Location) ([]Op, []Location) {
	v := uint8(0)
	if conditional {
		v = 1
	}
	return append(ops, buildOp(OpKindJump, v, 0, uint32(jumpDelta))),
		append(locations, loc)
}

func AppendMakeObject(kind ObjectKind, numArgs int, loc Location,
	ops []Op, locations []Location,
) ([]Op, []Location) {
	return append(ops, buildOp(OpKindMakeObject, uint8(kind), 0, uint32(numArgs))),
		append(locations, loc)

}

func AppendMakePattern(
	kind PatternKind, name string, numNested uint8,
	loc Location, ops []Op, locations []Location, binary *Binary, hash *BinaryHash,
) ([]Op, []Location) {
	return append(ops, buildOp(OpKindMakePattern, uint8(kind), numNested, uint32(hash.HashString(name, binary)))),
		append(locations, loc)
}

func AppendMakePatternLong(
	kind PatternKind, numNested uint32,
	loc Location, ops []Op, locations []Location, binary *Binary,
) ([]Op, []Location) {
	return append(ops, buildOp(OpKindMakePattern, uint8(kind), 0, numNested)),
		append(locations, loc)
}

func AppendAccess(filed string, loc Location, ops []Op, locations []Location, binary *Binary, hash *BinaryHash,
) ([]Op, []Location) {
	return append(ops, buildOp(OpKindAccess, 0, 0, uint32(hash.HashString(filed, binary)))),
		append(locations, loc)
}

func AppendUpdate(field string, loc Location,
	ops []Op, locations []Location, binary *Binary, hash *BinaryHash,
) ([]Op, []Location) {
	return append(ops, buildOp(OpKindUpdate, 0, 0, uint32(hash.HashString(field, binary)))),
		append(locations, loc)
}

func AppendSwapPop(loc Location, mode SwapPopMode,
	ops []Op, locations []Location,
) ([]Op, []Location) {
	return append(ops, buildOp(OpKindSwapPop, uint8(mode), 0, 0)),
		append(locations, loc)
}

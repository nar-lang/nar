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
}

// LoadLocal adds named local object to the top of the stack
type LoadLocal struct {
	Name StringHash
}

func (op LoadLocal) Word() uint64 {
	return uint64(OpKindLoadLocal) |
		(uint64(op.Name) << 32)
}

// LoadGlobal adds global object to the top of the stack
type LoadGlobal struct {
	Pointer Pointer
}

func (op LoadGlobal) Word() uint64 {
	return uint64(OpKindLoadGlobal) |
		(uint64(op.Pointer) << 32)
}

// LoadConst adds const value object to the top of the stack
type LoadConst struct {
	Stack StackKind
	Kind  ConstKind
	Value ConstHash
}

func (op LoadConst) Word() uint64 {
	return uint64(OpKindLoadConst) |
		(uint64(op.Stack) << 8) |
		(uint64(op.Kind) << 16) |
		(uint64(op.Value) << 32)
}

// Apply executes the function from the top of the stack.
// Arguments are taken from the top of the stack in reverse order
// (topmost object is the last arg). Returned value is left on the top of the stack.
// In case of NumArgs is less than number of function parameters it creates
// a closure and leaves it on the top of the stack
type Apply struct {
	NumArgs uint8
}

func (op Apply) Word() uint64 {
	return uint64(OpKindApply) |
		(uint64(op.NumArgs) << 8)
}

// Call executes an extern function.
// Arguments are taken from the top of the stack in reverse order
// (topmost object is last arg). Returned value is left on the top of the stack.
type Call struct {
	Name    StringHash
	NumArgs uint8
}

func (op Call) Word() uint64 {
	return uint64(OpKindCall) |
		(uint64(op.NumArgs) << 8) |
		(uint64(op.Name) << 32)
}

// Match tries to match pattern with object on the top of the stack.
// If it cannot be matched it moves on delta ops
// If it matches successfully - locals are extracted from pattern
// Matched object is left on the top of the stack in both cases
type Match struct {
	JumpDelta int32
}

func (op Match) Word() uint64 {
	return uint64(OpKindMatch) |
		(uint64(op.JumpDelta) << 32)
}

// Jump moves on delta ops unconditional
type Jump struct {
	Delta int32
}

func (op Jump) Word() uint64 {
	return uint64(OpKindJump) |
		(uint64(op.Delta) << 32)
}

// MakeObject creates an object on stack.
// List items stored on stack in reverse order (topmost object is the last item)
// Record fields stored as repeating pairs const string and value (field name is on the top of the stack)
// Data stores option name as const string on the top of the stack and
// args after that in reverse order (topmost is the last arg)
type MakeObject struct {
	Kind    ObjectKind
	NumArgs uint32
}

func (op MakeObject) Word() uint64 {
	return uint64(OpKindMakeObject) |
		(uint64(op.Kind) << 8) |
		(uint64(op.NumArgs) << 32)
}

// MakePattern creates pattern object
// Arguments are taken from the top of the stack in reverse order
// (topmost object is the last arg). Created object is left on the top of the stack.
type MakePattern struct {
	Kind      PatternKind
	Name      StringHash
	NumNested uint8
}

func (op MakePattern) Word() uint64 {
	return uint64(OpKindMakePattern) |
		(uint64(op.Kind) << 8) |
		(uint64(op.NumNested) << 16) |
		(uint64(op.Name) << 32)
}

// Access takes record object from the top of the stack and leaves its field on the stack
type Access struct {
	Field StringHash
}

func (op Access) Word() uint64 {
	return uint64(OpKindAccess) |
		(uint64(op.Field) << 32)
}

// Update create new record with replaced field from the top of the stack and rest fields
// form the second record object from stack. Created record is left on the top of the stack
type Update struct {
	Field StringHash
}

func (op Update) Word() uint64 {
	return uint64(OpKindUpdate) |
		(uint64(op.Field) << 32)
}

// SwapPop removes second object from the top of the stack and leaves first object on the top of the stack
type SwapPop struct {
	Mode SwapPopMode
}

func (op SwapPop) Word() uint64 {
	return uint64(OpKindSwapPop) |
		(uint64(op.Mode) << 8)
}

package bytecode

import "math"

type ConstHashKind uint8

const (
	ConstHashKindNone ConstHashKind = iota
	ConstHashKindInt
	ConstHashKindFloat
)

type PackedConst interface {
	Pack() uint64
	Kind() ConstHashKind
}

type PackedInt struct {
	Value int64
}

func (c PackedInt) Pack() uint64 {
	return uint64(c.Value)
}

func (c PackedInt) Kind() ConstHashKind {
	return ConstHashKindInt
}

type PackedFloat struct {
	Value float64
}

func (c PackedFloat) Pack() uint64 {
	return math.Float64bits(c.Value)
}

func (c PackedFloat) Kind() ConstHashKind {
	return ConstHashKindFloat
}

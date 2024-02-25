package ast

import (
	"fmt"
	"nar-compiler/pkg/bytecode"
)

type ConstValue interface {
	Coder
	EqualsTo(o ConstValue) bool
	AppendBytecode(stackKind bytecode.StackKind, loc Location, ops []bytecode.Op, locations []bytecode.Location, binary *bytecode.Binary, hash *bytecode.BinaryHash) ([]bytecode.Op, []bytecode.Location)
}

type CChar struct {
	Value rune
}

func (c CChar) AppendBytecode(stackKind bytecode.StackKind, loc Location, ops []bytecode.Op, locations []bytecode.Location, binary *bytecode.Binary, hash *bytecode.BinaryHash) ([]bytecode.Op, []bytecode.Location) {
	return bytecode.AppendLoadConstCharValue(c.Value, stackKind, loc.Bytecode(), ops, locations, binary)
}

func (c CChar) EqualsTo(o ConstValue) bool {
	if y, ok := o.(*CChar); ok {
		return c.Value == y.Value
	}
	return false
}

func (c CChar) Code(currentModule QualifiedIdentifier) string {
	return fmt.Sprintf("'%c'", c.Value)
}

type CInt struct {
	Value int64
}

func (c CInt) AppendBytecode(stackKind bytecode.StackKind, loc Location, ops []bytecode.Op, locations []bytecode.Location, binary *bytecode.Binary, hash *bytecode.BinaryHash) ([]bytecode.Op, []bytecode.Location) {
	return bytecode.AppendLoadConstIntValue(c.Value, stackKind, loc.Bytecode(), ops, locations, binary, hash)
}

func (c CInt) EqualsTo(o ConstValue) bool {
	if y, ok := o.(*CInt); ok {
		return c.Value == y.Value
	}
	return false
}

func (c CInt) Code(currentModule QualifiedIdentifier) string {
	return fmt.Sprintf("%d", c.Value)
}

type CFloat struct {
	Value float64
}

func (c CFloat) AppendBytecode(stackKind bytecode.StackKind, loc Location, ops []bytecode.Op, locations []bytecode.Location, binary *bytecode.Binary, hash *bytecode.BinaryHash) ([]bytecode.Op, []bytecode.Location) {
	return bytecode.AppendLoadConstFloatValue(c.Value, stackKind, loc.Bytecode(), ops, locations, binary, hash)
}

func (c CFloat) EqualsTo(o ConstValue) bool {
	if y, ok := o.(*CFloat); ok {
		return c.Value == y.Value
	}
	return false
}

func (c CFloat) Code(currentModule QualifiedIdentifier) string {
	return fmt.Sprintf("%f", c.Value)
}

type CString struct {
	Value string
}

func (c CString) AppendBytecode(stackKind bytecode.StackKind, loc Location, ops []bytecode.Op, locations []bytecode.Location, binary *bytecode.Binary, hash *bytecode.BinaryHash) ([]bytecode.Op, []bytecode.Location) {
	return bytecode.AppendLoadConstStringValue(c.Value, stackKind, loc.Bytecode(), ops, locations, binary, hash)
}

func (CString) _constValue() {}

func (c CString) EqualsTo(o ConstValue) bool {
	if y, ok := o.(*CString); ok {
		return c.Value == y.Value
	}
	return false
}

func (c CString) Code(currentModule QualifiedIdentifier) string {
	return fmt.Sprintf("\"%s\"", c.Value)
}

type CUnit struct {
}

func (c CUnit) AppendBytecode(stackKind bytecode.StackKind, loc Location, ops []bytecode.Op, locations []bytecode.Location, binary *bytecode.Binary, hash *bytecode.BinaryHash) ([]bytecode.Op, []bytecode.Location) {
	return bytecode.AppendLoadConstUnitValue(stackKind, loc.Bytecode(), ops, locations, binary)
}

func (CUnit) _constValue() {}

func (c CUnit) EqualsTo(o ConstValue) bool {
	_, ok := o.(*CUnit)
	return ok
}

func (c CUnit) Code(currentModule QualifiedIdentifier) string {
	return "()"
}

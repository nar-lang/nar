package ast

import "fmt"

type ConstValue interface {
	Coder
	EqualsTo(o ConstValue) bool
}

type CChar struct {
	Value rune
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

func (CUnit) _constValue() {}

func (c CUnit) EqualsTo(o ConstValue) bool {
	_, ok := o.(*CUnit)
	return ok
}

func (c CUnit) Code(currentModule QualifiedIdentifier) string {
	return "()"
}

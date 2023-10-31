package ast

import "fmt"

type ConstValue interface {
	fmt.Stringer
	_constValue()
}

type CChar struct {
	Value rune
}

func (CChar) _constValue() {}

func (c CChar) String() string {
	return fmt.Sprintf("CChar(%v)", c.Value)
}

type CInt struct {
	Value int64
}

func (CInt) _constValue() {}

func (c CInt) String() string {
	return fmt.Sprintf("CInt(%v)", c.Value)
}

type CFloat struct {
	Value float64
}

func (CFloat) _constValue() {}

func (c CFloat) String() string {
	return fmt.Sprintf("CFloat(%v)", c.Value)
}

type CString struct {
	Value string
}

func (CString) _constValue() {}

func (c CString) String() string {
	return fmt.Sprintf("CString(%v)", c.Value)
}

type CUnit struct {
}

func (CUnit) _constValue() {}

func (c CUnit) String() string {
	return fmt.Sprintf("CUnit()")
}

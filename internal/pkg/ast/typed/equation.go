package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
)

type Equation struct {
	left, right Type
	stmt        Statement
}

func NewEquationBestLoc(left Type, right Type, enclosing ast.Location) Equation {
	var stmt Statement
	if enclosing.Contains(left.Location()) {
		stmt = left
	} else if enclosing.Contains(right.Location()) {
		stmt = right
	} else {
		stmt = NewDefinition(enclosing, 0, false, "---", enclosing)
	}

	return Equation{
		left:  left,
		right: right,
		stmt:  stmt,
	}
}

func NewEquation(stmt Statement, left Type, right Type) Equation {
	return Equation{
		left:  left,
		right: right,
		stmt:  stmt,
	}
}

func (eq Equation) String(index int) string {
	return fmt.Sprintf(
		"| %d | %s | `%s` | `%s` | `%s` |\n",
		index, eq.stmt.Location().CursorString(), eq.left.Code(""), eq.right.Code(""), eq.stmt.Code(""))
}

func (eq Equation) equalsTo(other Equation) bool {
	return (eq.left.equalsTo(other.left, nil) && eq.right.equalsTo(other.right, nil)) ||
		(eq.right.equalsTo(other.left, nil) && eq.left.equalsTo(other.right, nil))
}

func (eq Equation) isRedundant() bool {
	return eq.left.equalsTo(eq.right, nil)
}

type Equations []Equation

func (eqs Equations) String() string {
	var s string
	for i, eq := range eqs {
		s += eq.String(i)
	}
	return s
}

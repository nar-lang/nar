package typed

import (
	"fmt"
	"oak-compiler/ast"
)

type Definition struct {
	Id         uint64
	Type       Type
	Pattern    Pattern
	Expression Expression
}

func (d *Definition) String() string {
	return fmt.Sprintf("Def(%v,%v)", d.Pattern, d.Expression)
}

type Module struct {
	Path        string
	Definitions map[ast.Identifier]*Definition
}

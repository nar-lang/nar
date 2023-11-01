package normalized

import (
	"oak-compiler/ast"
)

type Definition struct {
	Id         uint64
	Pattern    Pattern
	Expression Expression
	Type       Type
}

type Module struct {
	Path        string
	DepPaths    []string
	Definitions map[ast.Identifier]Definition
}

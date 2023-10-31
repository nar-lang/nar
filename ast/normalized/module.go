package normalized

import (
	"oak-compiler/ast"
)

type Definition struct {
	Pattern    Pattern
	Expression Expression
	Type       Type
}

type Module struct {
	Path        string
	DepPaths    []string
	Definitions map[ast.Identifier]Definition
}

package normalized

import (
	"oak-compiler/ast"
)

type Definition struct {
	Id         uint64
	Pattern    Pattern
	Expression Expression
	Type       Type
	Location   ast.Location
	Hidden     bool
}

type Module struct {
	Path        string
	Name        ast.QualifiedIdentifier
	DepPaths    []string
	Definitions map[ast.Identifier]Definition
}

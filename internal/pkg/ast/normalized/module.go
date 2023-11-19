package normalized

import (
	"oak-compiler/internal/pkg/ast"
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
	Name         ast.QualifiedIdentifier
	Dependencies []ast.QualifiedIdentifier
	Definitions  map[ast.Identifier]Definition
}

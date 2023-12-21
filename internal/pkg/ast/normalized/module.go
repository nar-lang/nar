package normalized

import (
	"oak-compiler/internal/pkg/ast"
)

type Definition struct {
	Id         uint64
	Name       ast.Identifier
	Params     []Pattern
	Expression Expression
	Type       Type
	Location   ast.Location
	Hidden     bool
}

type Module struct {
	Name         ast.QualifiedIdentifier
	Dependencies map[ast.QualifiedIdentifier][]ast.Identifier
	Definitions  []*Definition
}

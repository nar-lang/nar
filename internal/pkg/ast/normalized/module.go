package normalized

import (
	"nar-compiler/internal/pkg/ast"
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
	Location     ast.Location
	Dependencies map[ast.QualifiedIdentifier][]ast.Identifier
	Definitions  []*Definition
}

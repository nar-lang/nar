package parsed

import "nar-compiler/internal/pkg/ast"

type Definition struct {
	Location   ast.Location
	Hidden     bool
	Name       ast.Identifier
	Params     []Pattern
	Expression Expression
	Type       Type
}

func (def *Definition) GetLocation() ast.Location {
	return def.Location
}

func (def *Definition) _parsed() {}

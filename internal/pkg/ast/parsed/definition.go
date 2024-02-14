package parsed

import "nar-compiler/internal/pkg/ast"

type Definition struct {
	location   ast.Location
	hidden     bool
	name       ast.Identifier
	params     []Pattern
	expression Expression
	type_      Type
}

func NewDefinition(
	location ast.Location,
	hidden bool,
	name ast.Identifier,
	params []Pattern,
	expression Expression,
	type_ Type,
) *Definition {
	return &Definition{
		location:   location,
		hidden:     hidden,
		name:       name,
		params:     params,
		expression: expression,
		type_:      type_,
	}
}

func (def *Definition) GetLocation() ast.Location {
	return def.location
}

func (def *Definition) _parsed() {}

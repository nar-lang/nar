package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
)

type Statement interface {
	Location() ast.Location
}

type placeholderMap map[ast.FullIdentifier]typed.Type

type typeParamsMap map[ast.Identifier]typed.Type

var LastDefinitionId = uint64(0)

var lastLambdaId = uint64(0)

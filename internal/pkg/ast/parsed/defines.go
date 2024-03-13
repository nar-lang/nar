package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

type Statement interface {
	Location() ast.Location
	Iterate(f func(statement Statement))
	Successor() normalized.Statement
	_parsed()
	SemanticTokens() []ast.SemanticToken
}

type namedTypeMap map[ast.FullIdentifier]*normalized.TPlaceholder

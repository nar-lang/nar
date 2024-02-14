package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

type Statement interface {
	GetLocation() ast.Location
	_parsed()
}

type namedTypeMap map[ast.FullIdentifier]*normalized.TPlaceholder

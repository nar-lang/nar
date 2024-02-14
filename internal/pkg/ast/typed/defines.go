package typed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/bytecode"
)

type Statement interface {
	ast.Coder
	Location() ast.Location
	Children() []Statement
}

type bytecoder interface {
	appendBytecode(ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary) ([]bytecode.Op, []ast.Location)
}

type localTypesMap map[ast.Identifier]Type

type TypePredecessor interface {
	SetSuccessor(Type) Type
}

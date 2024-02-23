package typed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/pkg/bytecode"
)

type Statement interface {
	ast.Coder
	Location() ast.Location
	Children() []Statement
}

type bytecoder interface {
	appendBytecode(ops []bytecode.Op, locations []bytecode.Location, binary *bytecode.Binary) ([]bytecode.Op, []bytecode.Location)
}

type localTypesMap map[ast.Identifier]Type

type TypePredecessor interface {
	SetSuccessor(Type) Type
}

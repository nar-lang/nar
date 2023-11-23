package parsed

import (
	"oak-compiler/internal/pkg/ast"
)

type Import struct {
	Location         ast.Location
	ModuleIdentifier ast.QualifiedIdentifier
	Alias            *ast.Identifier
	ExposingAll      bool
	Exposing         []string
}

type Alias struct {
	Location ast.Location
	Hidden   bool
	Name     ast.Identifier
	Params   []ast.Identifier
	Type     Type
}

type Associativity int

const (
	Left  Associativity = -1
	None                = 0
	Right               = 1
)

type Infix struct {
	Location      ast.Location
	Hidden        bool
	Name          ast.InfixIdentifier
	Associativity Associativity
	Precedence    int
	AliasLocation ast.Location
	Alias         ast.Identifier
}

type Definition struct {
	Location   ast.Location
	Hidden     bool
	Name       ast.Identifier
	Params     []Pattern
	Expression Expression
	Type       Type
}

type DataTypeOption struct {
	Location ast.Location
	Hidden   bool
	Name     ast.Identifier
	Params   []Type
}

type DataType struct {
	Location ast.Location
	Hidden   bool
	Name     ast.Identifier
	Params   []ast.Identifier
	Options  []DataTypeOption
}

type Module struct {
	Name        ast.QualifiedIdentifier
	Imports     []Import
	Aliases     []Alias
	InfixFns    []Infix
	Definitions []Definition
	DataTypes   []DataType
}

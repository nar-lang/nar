package parsed

import "nar-compiler/internal/pkg/ast"

type Statement interface {
	GetLocation() ast.Location
	_parsed()
}

type RecordField struct {
	Location ast.Location
	Name     ast.Identifier
	Value    Expression
}

type SelectCase struct {
	ast.Location
	Pattern    Pattern
	Expression Expression
}

type BinOpItem struct {
	Expression Expression
	Infix      ast.InfixIdentifier
	Fn         *Infix
}

type DataOption struct {
	name   ast.Identifier
	hidden bool
	values []Type
}

func NewDataOption(name ast.Identifier, hidden bool, values []Type) DataOption {
	return DataOption{
		name:   name,
		hidden: hidden,
		values: values,
	}
}

package parsed

import (
	"nar-compiler/internal/pkg/ast"
)

type Expression interface {
	_expression()
	GetLocation() ast.Location
}

type Access struct {
	ast.Location
	Record    Expression
	FieldName ast.Identifier
}

func (Access) _expression() {}

func (e Access) GetLocation() ast.Location {
	return e.Location
}

type Apply struct {
	ast.Location
	Func Expression
	Args []Expression
}

func (Apply) _expression() {}

func (e Apply) GetLocation() ast.Location {
	return e.Location
}

type Const struct {
	ast.Location
	Value ast.ConstValue
}

func (Const) _expression() {}

func (e Const) GetLocation() ast.Location {
	return e.Location
}

type If struct {
	ast.Location
	Condition, Positive, Negative Expression
}

func (If) _expression() {}

func (e If) GetLocation() ast.Location {
	return e.Location
}

type LetMatch struct {
	ast.Location
	Pattern Pattern
	Value   Expression
	Nested  Expression
}

func (LetMatch) _expression() {}

func (e LetMatch) GetLocation() ast.Location {
	return e.Location
}

type LetDef struct {
	ast.Location
	Name   ast.Identifier
	Params []Pattern
	Body   Expression
	FnType Type
	Nested Expression
}

func (LetDef) _expression() {}

func (e LetDef) GetLocation() ast.Location {
	return e.Location
}

type List struct {
	ast.Location
	Items []Expression
}

func (List) _expression() {}

func (e List) GetLocation() ast.Location {
	return e.Location
}

type RecordField struct {
	ast.Location
	Name  ast.Identifier
	Value Expression
}

type Record struct {
	ast.Location
	Fields []RecordField
}

func (Record) _expression() {}

func (e Record) GetLocation() ast.Location {
	return e.Location
}

type SelectCase struct {
	ast.Location
	Pattern    Pattern
	Expression Expression
}

type Select struct {
	ast.Location
	Condition Expression
	Cases     []SelectCase
}

func (Select) _expression() {}

func (e Select) GetLocation() ast.Location {
	return e.Location
}

type Tuple struct {
	ast.Location
	Items []Expression
}

func (Tuple) _expression() {}

func (e Tuple) GetLocation() ast.Location {
	return e.Location
}

type Update struct {
	ast.Location
	RecordName ast.QualifiedIdentifier
	Fields     []RecordField
}

func (Update) _expression() {}

func (e Update) GetLocation() ast.Location {
	return e.Location
}

type Lambda struct {
	ast.Location
	Params []Pattern
	Return Type
	Body   Expression
}

func (Lambda) _expression() {}

func (e Lambda) GetLocation() ast.Location {
	return e.Location
}

type Accessor struct {
	ast.Location
	FieldName ast.Identifier
}

func (Accessor) _expression() {}

func (e Accessor) GetLocation() ast.Location {
	return e.Location
}

type BinOpItem struct {
	Expression Expression
	Infix      ast.InfixIdentifier
	Fn         Infix
}

type BinOp struct {
	ast.Location
	Items         []BinOpItem
	InParentheses bool
}

func (BinOp) _expression() {}

func (e BinOp) GetLocation() ast.Location {
	return e.Location
}

type Negate struct {
	ast.Location
	Nested Expression
}

func (Negate) _expression() {}

func (e Negate) GetLocation() ast.Location {
	return e.Location
}

type Var struct {
	ast.Location
	Name ast.QualifiedIdentifier
}

func (Var) _expression() {}

func (e Var) GetLocation() ast.Location {
	return e.Location
}

type Constructor struct {
	ast.Location
	ModuleName ast.QualifiedIdentifier
	DataName   ast.Identifier
	OptionName ast.Identifier
	Args       []Expression
}

func (Constructor) _expression() {}

func (e Constructor) GetLocation() ast.Location {
	return e.Location
}

type InfixVar struct {
	ast.Location
	Infix ast.InfixIdentifier
}

func (InfixVar) _expression() {}

func (e InfixVar) GetLocation() ast.Location {
	return e.Location
}

type NativeCall struct {
	Location ast.Location
	Name     ast.FullIdentifier
	Args     []Expression
}

func (NativeCall) _expression() {}

func (e NativeCall) GetLocation() ast.Location {
	return e.Location
}

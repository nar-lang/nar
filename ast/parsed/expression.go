package parsed

import (
	"oak-compiler/ast"
)

type Expression interface {
	_expression()
}

type Access struct {
	ast.Location
	Record    Expression
	FieldName ast.Identifier
}

func (Access) _expression() {}

type Call struct {
	ast.Location
	Func Expression
	Args []Expression
}

func (Call) _expression() {}

type Const struct {
	ast.Location
	Value ast.ConstValue
}

func (Const) _expression() {}

type If struct {
	ast.Location
	Condition, Positive, Negative Expression
}

func (If) _expression() {}

type Let struct {
	ast.Location
	Definition Definition
	Body       Expression
}

func (Let) _expression() {}

type List struct {
	ast.Location
	Items []Expression
}

func (List) _expression() {}

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

type Tuple struct {
	ast.Location
	Items []Expression
}

func (Tuple) _expression() {}

type Update struct {
	ast.Location
	RecordName ast.QualifiedIdentifier
	Fields     []RecordField
}

func (Update) _expression() {}

type Lambda struct {
	ast.Location
	Params []Pattern
	Return Type
	Body   Expression
}

func (Lambda) _expression() {}

type Accessor struct {
	ast.Location
	FieldName ast.Identifier
}

func (Accessor) _expression() {}

type BinOp struct {
	ast.Location
	Infix       ast.InfixIdentifier
	Left, Right Expression
}

func (BinOp) _expression() {}

type Negate struct {
	ast.Location
	Nested Expression
}

func (Negate) _expression() {}

type Var struct {
	ast.Location
	Name ast.QualifiedIdentifier
}

func (Var) _expression() {}

type Constructor struct {
	ast.Location
	DataName  ast.ExternalIdentifier
	ValueName ast.Identifier
	Args      []Expression
}

func (Constructor) _expression() {}

type InfixVar struct {
	ast.Location
	Infix ast.InfixIdentifier
}

func (InfixVar) _expression() {}

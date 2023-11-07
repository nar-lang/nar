package normalized

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

type UpdateLocal struct {
	ast.Location
	RecordName ast.Identifier
	Fields     []RecordField
}

func (UpdateLocal) _expression() {}

type UpdateGlobal struct {
	ast.Location
	ModulePath     string
	DefinitionName ast.Identifier
	Fields         []RecordField
}

func (UpdateGlobal) _expression() {}

type Lambda struct {
	ast.Location
	Params []Pattern
	Body   Expression
}

func (Lambda) _expression() {}

type Constructor struct {
	ast.Location
	DataName  ast.ExternalIdentifier
	ValueName ast.Identifier
	Args      []Expression
}

func (Constructor) _expression() {}

type NativeCall struct {
	ast.Location
	Name ast.ExternalIdentifier
	Args []Expression
}

func (NativeCall) _expression() {}

type Var struct {
	ast.Location
	Name           ast.QualifiedIdentifier
	ModulePath     string
	DefinitionName ast.Identifier
}

func (Var) _expression() {}

package normalized

import (
	"nar-compiler/internal/pkg/ast"
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

type Apply struct {
	ast.Location
	Func Expression
	Args []Expression
}

func (Apply) _expression() {}

type Const struct {
	ast.Location
	Value ast.ConstValue
}

func (Const) _expression() {}

type LetMatch struct {
	ast.Location
	Pattern Pattern
	Value   Expression
	Nested  Expression
}

func (LetMatch) _expression() {}

type LetDef struct {
	ast.Location
	Name   ast.Identifier
	Params []Pattern
	Body   Expression
	FnType Type
	Nested Expression
}

func (LetDef) _expression() {}

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
	ModuleName     ast.QualifiedIdentifier
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
	ModuleName ast.QualifiedIdentifier
	DataName   ast.Identifier
	OptionName ast.Identifier
	Args       []Expression
}

func (Constructor) _expression() {}

type NativeCall struct {
	ast.Location
	Name ast.FullIdentifier
	Args []Expression
}

func (NativeCall) _expression() {}

type Local struct {
	ast.Location
	Name ast.Identifier
}

func (Local) _expression() {}

type Global struct {
	ast.Location
	ModuleName     ast.QualifiedIdentifier
	DefinitionName ast.Identifier
}

func (Global) _expression() {}

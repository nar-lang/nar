package normalized

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
)

type Expression interface {
	_expression()
	GetPredecessor() ExpressionWithSuccessor
	SetPredecessor(expr ExpressionWithSuccessor)
}

type ExpressionWithSuccessor interface {
	SetSuccessor(expr Expression) Expression
}

type ExpressionBase struct {
	Location    ast.Location
	predecessor ExpressionWithSuccessor
	successor   typed.Expression
}

func (e *ExpressionBase) _expression() {}

func (e *ExpressionBase) GetLocation() ast.Location {
	return e.Location
}

func (e *ExpressionBase) GetPredecessor() ExpressionWithSuccessor {
	return e.predecessor
}

func (e *ExpressionBase) SetPredecessor(expr ExpressionWithSuccessor) {
	e.predecessor = expr
}

func (e *ExpressionBase) GetSuccessor() typed.Expression {
	return e.successor
}

func (e *ExpressionBase) SetSuccessor(expr typed.Expression) typed.Expression {
	e.successor = expr
	return expr
}

type Access struct {
	*ExpressionBase
	Record    Expression
	FieldName ast.Identifier
}

type Apply struct {
	*ExpressionBase
	Func Expression
	Args []Expression
}

type Const struct {
	*ExpressionBase
	Value ast.ConstValue
}

type LetMatch struct {
	*ExpressionBase
	Pattern Pattern
	Value   Expression
	Nested  Expression
}

type LetDef struct {
	*ExpressionBase
	Name   ast.Identifier
	Params []Pattern
	Body   Expression
	FnType Type
	Nested Expression
}

type List struct {
	*ExpressionBase
	Items []Expression
}

type RecordField struct {
	ast.Location
	Name  ast.Identifier
	Value Expression
}

type Record struct {
	*ExpressionBase
	Fields []RecordField
}

type SelectCase struct {
	ast.Location
	Pattern    Pattern
	Expression Expression
}

type Select struct {
	*ExpressionBase
	Condition Expression
	Cases     []SelectCase
}

type Tuple struct {
	*ExpressionBase
	Items []Expression
}

type UpdateLocal struct {
	*ExpressionBase
	RecordName ast.Identifier
	Fields     []RecordField
}

type UpdateGlobal struct {
	*ExpressionBase
	ModuleName     ast.QualifiedIdentifier
	DefinitionName ast.Identifier
	Fields         []RecordField
}

type Lambda struct {
	*ExpressionBase
	Params []Pattern
	Body   Expression
}

type Constructor struct {
	*ExpressionBase
	ModuleName ast.QualifiedIdentifier
	DataName   ast.Identifier
	OptionName ast.Identifier
	Args       []Expression
}

type NativeCall struct {
	*ExpressionBase
	Name ast.FullIdentifier
	Args []Expression
}

type Local struct {
	*ExpressionBase
	Name   ast.Identifier
	Target Pattern
}

type Global struct {
	*ExpressionBase
	ModuleName     ast.QualifiedIdentifier
	DefinitionName ast.Identifier
}

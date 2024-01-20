package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

type Expression interface {
	Statement
	GetSuccessor() normalized.Expression
	normalize(
		locals map[ast.Identifier]normalized.Pattern,
		modules map[ast.QualifiedIdentifier]*Module,
		module *Module,
		normalizedModule *normalized.Module,
	) (normalized.Expression, error)
	setSuccessor(expr normalized.Expression)
}

type ExpressionBase struct {
	Location  ast.Location
	successor normalized.Expression
}

func (*ExpressionBase) _parsed() {}

func (e *ExpressionBase) GetLocation() ast.Location {
	return e.Location
}

func (e *ExpressionBase) GetSuccessor() normalized.Expression {
	return e.successor
}

func (e *ExpressionBase) setSuccessor(expr normalized.Expression) {
	e.successor = expr
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

type If struct {
	*ExpressionBase
	Condition, Positive, Negative Expression
}

type LetMatch struct {
	*ExpressionBase
	Pattern Pattern
	Value   Expression
	Nested  Expression
}

type LetDef struct {
	*ExpressionBase
	Name         ast.Identifier
	NameLocation ast.Location
	Params       []Pattern
	Body         Expression
	FnType       Type
	Nested       Expression
}

type List struct {
	*ExpressionBase
	Items []Expression
}

type RecordField struct {
	Location ast.Location
	Name     ast.Identifier
	Value    Expression
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

type Update struct {
	*ExpressionBase
	RecordName ast.QualifiedIdentifier
	Fields     []RecordField
}

type Lambda struct {
	*ExpressionBase
	Params []Pattern
	Return Type
	Body   Expression
}

type Accessor struct {
	*ExpressionBase
	FieldName ast.Identifier
}

type BinOpItem struct {
	Expression Expression
	Infix      ast.InfixIdentifier
	Fn         *Infix
}

type BinOp struct {
	*ExpressionBase
	Items         []BinOpItem
	InParentheses bool
}

type Negate struct {
	*ExpressionBase
	Nested Expression
}

type Var struct {
	*ExpressionBase
	Name ast.QualifiedIdentifier
}

type Constructor struct {
	*ExpressionBase
	ModuleName ast.QualifiedIdentifier
	DataName   ast.Identifier
	OptionName ast.Identifier
	Args       []Expression
}

type InfixVar struct {
	*ExpressionBase
	Infix ast.InfixIdentifier
}

type NativeCall struct {
	*ExpressionBase
	Name ast.FullIdentifier
	Args []Expression
}

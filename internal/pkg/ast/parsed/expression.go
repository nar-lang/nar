package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

type Expression interface {
	Statement
	normalize(
		locals map[ast.Identifier]normalized.Pattern,
		modules map[ast.QualifiedIdentifier]*Module,
		module *Module,
		normalizedModule *normalized.Module,
	) (normalized.Expression, error)
	Successor() normalized.Expression
	setSuccessor(expr normalized.Expression)
}

type expressionBase struct {
	location  ast.Location
	successor normalized.Expression
}

func (*expressionBase) _parsed() {}

func (e *expressionBase) GetLocation() ast.Location {
	return e.location
}

func (e *expressionBase) Successor() normalized.Expression {
	return e.successor
}

func (e *expressionBase) setSuccessor(expr normalized.Expression) {
	e.successor = expr
}

func newExpressionBase(location ast.Location) *expressionBase {
	return &expressionBase{location: location}
}

type Access struct {
	*expressionBase
	record    Expression
	fieldName ast.Identifier
}

func NewAccess(location ast.Location, record Expression, fieldName ast.Identifier) Expression {
	return &Access{
		expressionBase: newExpressionBase(location),
		record:         record,
		fieldName:      fieldName,
	}
}

type Apply struct {
	*expressionBase
	func_ Expression
	args  []Expression
}

func NewApply(location ast.Location, function Expression, args []Expression) Expression {
	return &Apply{
		expressionBase: newExpressionBase(location),
		func_:          function,
		args:           args,
	}
}

type Const struct {
	*expressionBase
	value ast.ConstValue
}

func NewConst(location ast.Location, value ast.ConstValue) Expression {
	return &Const{
		expressionBase: newExpressionBase(location),
		value:          value,
	}
}

type If struct {
	*expressionBase
	condition, positive, negative Expression
}

func NewIf(location ast.Location, condition, positive, negative Expression) Expression {
	return &If{
		expressionBase: newExpressionBase(location),
		condition:      condition,
		positive:       positive,
		negative:       negative,
	}
}

type LetMatch struct {
	*expressionBase
	pattern Pattern
	value   Expression
	nested  Expression
}

func NewLetMatch(location ast.Location, pattern Pattern, value, nested Expression) Expression {
	return &LetMatch{
		expressionBase: newExpressionBase(location),
		pattern:        pattern,
		value:          value,
		nested:         nested,
	}
}

type LetDef struct {
	*expressionBase
	name         ast.Identifier
	nameLocation ast.Location
	params       []Pattern
	body         Expression
	fnType       Type
	nested       Expression
}

func NewLetDef(
	location ast.Location, name ast.Identifier, nameLocation ast.Location,
	params []Pattern, body Expression, fnType Type, nested Expression,
) Expression {
	return &LetDef{
		expressionBase: newExpressionBase(location),
		name:           name,
		nameLocation:   nameLocation,
		params:         params,
		body:           body,
		fnType:         fnType,
		nested:         nested,
	}
}

type List struct {
	*expressionBase
	items []Expression
}

func NewList(location ast.Location, items []Expression) Expression {
	return &List{
		expressionBase: newExpressionBase(location),
		items:          items,
	}
}

type Record struct {
	*expressionBase
	fields []RecordField
}

func NewRecord(location ast.Location, fields []RecordField) Expression {
	return &Record{
		expressionBase: newExpressionBase(location),
		fields:         fields,
	}
}

type Select struct {
	*expressionBase
	condition Expression
	cases     []SelectCase
}

func NewSelect(location ast.Location, condition Expression, cases []SelectCase) Expression {
	return &Select{
		expressionBase: newExpressionBase(location),
		condition:      condition,
		cases:          cases,
	}
}

type Tuple struct {
	*expressionBase
	items []Expression
}

func NewTuple(location ast.Location, items []Expression) Expression {
	return &Tuple{
		expressionBase: newExpressionBase(location),
		items:          items,
	}
}

type Update struct {
	*expressionBase
	recordName ast.QualifiedIdentifier
	fields     []RecordField
}

func NewUpdate(location ast.Location, recordName ast.QualifiedIdentifier, fields []RecordField) Expression {
	return &Update{
		expressionBase: newExpressionBase(location),
		recordName:     recordName,
		fields:         fields,
	}
}

type Lambda struct {
	*expressionBase
	params  []Pattern
	return_ Type
	body    Expression
}

func NewLambda(location ast.Location, params []Pattern, returnType Type, body Expression) Expression {
	return &Lambda{
		expressionBase: newExpressionBase(location),
		params:         params,
		return_:        returnType,
		body:           body,
	}
}

type Accessor struct {
	*expressionBase
	fieldName ast.Identifier
}

func NewAccessor(location ast.Location, fieldName ast.Identifier) Expression {
	return &Accessor{
		expressionBase: newExpressionBase(location),
		fieldName:      fieldName,
	}
}

type BinOp struct {
	*expressionBase
	items         []BinOpItem
	inParentheses bool
}

func (e *BinOp) SetInParentheses(inParentheses bool) {
	e.inParentheses = inParentheses
}

func (e *BinOp) InParentheses() bool {
	return e.inParentheses
}

func (e *BinOp) Items() []BinOpItem {
	return e.items
}

func NewBinOp(location ast.Location, items []BinOpItem, inParentheses bool) Expression {
	return &BinOp{
		expressionBase: newExpressionBase(location),
		items:          items,
		inParentheses:  inParentheses,
	}
}

type Negate struct {
	*expressionBase
	nested Expression
}

func NewNegate(location ast.Location, nested Expression) Expression {
	return &Negate{
		expressionBase: newExpressionBase(location),
		nested:         nested,
	}
}

type Var struct {
	*expressionBase
	name ast.QualifiedIdentifier
}

func NewVar(location ast.Location, name ast.QualifiedIdentifier) Expression {
	return &Var{
		expressionBase: newExpressionBase(location),
		name:           name,
	}
}

type Constructor struct {
	*expressionBase
	moduleName ast.QualifiedIdentifier
	dataName   ast.Identifier
	optionName ast.Identifier
	args       []Expression
}

func NewConstructor(
	location ast.Location,
	moduleName ast.QualifiedIdentifier,
	dataName ast.Identifier,
	optionName ast.Identifier,
	args []Expression,
) Expression {
	return &Constructor{
		expressionBase: newExpressionBase(location),
		moduleName:     moduleName,
		dataName:       dataName,
		optionName:     optionName,
		args:           args,
	}
}

type InfixVar struct {
	*expressionBase
	infix ast.InfixIdentifier
}

func NewInfixVar(location ast.Location, infix ast.InfixIdentifier) Expression {
	return &InfixVar{
		expressionBase: newExpressionBase(location),
		infix:          infix,
	}
}

type NativeCall struct {
	*expressionBase
	name ast.FullIdentifier
	args []Expression
}

func NewNativeCall(location ast.Location, name ast.FullIdentifier, args []Expression) Expression {
	return &NativeCall{
		expressionBase: newExpressionBase(location),
		name:           name,
		args:           args,
	}
}

package parsed

import (
	"nar-compiler/internal/pkg/ast"
)

func FoldModule[T any](
	fe func(Expression, T) T, ft func(Type, T) T, fp func(Pattern, T) T, acc T, module *Module,
) T {
	if module == nil {
		return acc
	}

	for _, def := range module.definitions {
		acc = FoldDefinition(fe, ft, fp, acc, def)
	}
	for _, alias := range module.aliases {
		acc = FoldType(ft, acc, alias.type_)
	}
	for _, infix := range module.infixFns {
		acc = FoldExpression(fe, ft, fp, acc, NewVar(infix.aliasLocation, ast.QualifiedIdentifier(infix.alias)))
	}
	return acc
}

func FoldDefinition[T any](
	fe func(Expression, T) T, ft func(Type, T) T, fp func(Pattern, T) T, acc T, def *Definition,
) T {
	if def == nil {
		return acc
	}

	for _, p := range def.params {
		acc = FoldPattern(ft, fp, acc, p)
	}
	acc = FoldType(ft, acc, def.declaredType)
	acc = FoldExpression(fe, ft, fp, acc, def.expression)
	return acc
}

func FoldPattern[T any](
	ft func(Type, T) T, fp func(Pattern, T) T, acc T, pattern Pattern,
) T {
	if pattern == nil {
		return acc
	}

	acc = fp(pattern, acc)

	switch pattern.(type) {
	case *PAlias:
		{
			p := pattern.(*PAlias)
			acc = FoldPattern(ft, fp, acc, p.nested)
			acc = FoldType(ft, acc, p.declaredType)
		}
	case *PAny:
		{
			p := pattern.(*PAny)
			acc = FoldType(ft, acc, p.declaredType)
		}
	case *PCons:
		{
			p := pattern.(*PCons)
			acc = FoldPattern(ft, fp, acc, p.head)
			acc = FoldPattern(ft, fp, acc, p.tail)
			acc = FoldType(ft, acc, p.declaredType)
		}
	case *PConst:
		{
			p := pattern.(*PConst)
			acc = FoldType(ft, acc, p.declaredType)
		}
	case *POption:
		{
			p := pattern.(*POption)
			for _, a := range p.values {
				acc = FoldPattern(ft, fp, acc, a)
			}
			acc = FoldType(ft, acc, p.declaredType)
		}
	case *PList:
		{
			p := pattern.(*PList)
			for _, a := range p.items {
				acc = FoldPattern(ft, fp, acc, a)
			}
			acc = FoldType(ft, acc, p.declaredType)
		}
	case *PNamed:
		{
			p := pattern.(*PNamed)
			acc = FoldType(ft, acc, p.declaredType)
		}
	case *PRecord:
		{
			p := pattern.(*PRecord)
			acc = FoldType(ft, acc, p.declaredType)
		}
	case *PTuple:
		{
			p := pattern.(*PTuple)
			for _, f := range p.items {
				acc = FoldPattern(ft, fp, acc, f)
			}
		}
	default:
		panic("unreachable")
	}

	return acc
}

func FoldType[T any](
	ft func(Type, T) T, acc T, type_ Type,
) T {
	if type_ == nil {
		return acc
	}

	acc = ft(type_, acc)

	switch type_.(type) {
	case *TFunc:
		{
			t := type_.(*TFunc)
			for _, f := range t.params {
				acc = FoldType(ft, acc, f)
			}
			acc = FoldType(ft, acc, t.return_)
		}
	case *TRecord:
		{
			t := type_.(*TRecord)
			for _, f := range t.fields {
				acc = FoldType(ft, acc, f)
			}
		}
	case *TTuple:
		{
			t := type_.(*TTuple)
			for _, f := range t.items {
				acc = FoldType(ft, acc, f)
			}
		}
	case *TUnit:
		{

		}
	case *TNamed:
		{
			t := type_.(*TNamed)
			for _, f := range t.args {
				acc = FoldType(ft, acc, f)
			}
		}
	case *TData:
		{
			t := type_.(*TData)
			for _, f := range t.options {
				for _, v := range f.values {
					acc = FoldType(ft, acc, v)
				}
			}
		}
	case *TNative:
		{
			t := type_.(*TNative)
			for _, f := range t.args {
				acc = FoldType(ft, acc, f)
			}
		}
	case *TParameter:
		{

		}
	default:
		panic("unreachable")
	}

	return acc
}

func FoldExpression[T any](
	fe func(Expression, T) T, ft func(Type, T) T, fp func(Pattern, T) T, acc T, expr Expression,
) T {
	if expr == nil {
		return acc
	}

	acc = fe(expr, acc)

	switch expr.(type) {
	case *Access:
		{
			e := expr.(*Access)
			acc = FoldExpression(fe, ft, fp, acc, e.record)
		}
	case *Apply:
		{
			e := expr.(*Apply)
			acc = FoldExpression(fe, ft, fp, acc, e.func_)
			for _, a := range e.args {
				acc = FoldExpression(fe, ft, fp, acc, a)
			}
		}
	case *Const:
		{

		}
	case *If:
		{
			e := expr.(*If)
			acc = FoldExpression(fe, ft, fp, acc, e.condition)
			acc = FoldExpression(fe, ft, fp, acc, e.positive)
			acc = FoldExpression(fe, ft, fp, acc, e.negative)
		}
	case *Let:
		{
			e := expr.(*Let)
			acc = FoldExpression(fe, ft, fp, acc, e.value)
			acc = FoldExpression(fe, ft, fp, acc, e.nested)
			acc = FoldPattern(ft, fp, acc, e.pattern)
		}
	case *Function:
		{
			e := expr.(*Function)
			for _, p := range e.params {
				acc = FoldPattern(ft, fp, acc, p)
			}
			acc = FoldExpression(fe, ft, fp, acc, e.body)
			acc = FoldType(ft, acc, e.declaredType)
			acc = FoldExpression(fe, ft, fp, acc, e.nested)
		}
	case *List:
		{
			e := expr.(*List)
			for _, a := range e.items {
				acc = FoldExpression(fe, ft, fp, acc, a)
			}
		}
	case *Record:
		{
			e := expr.(*Record)
			for _, a := range e.fields {
				acc = FoldExpression(fe, ft, fp, acc, a.value)
			}
		}
	case *Select:
		{
			e := expr.(*Select)
			acc = FoldExpression(fe, ft, fp, acc, e.condition)
			for _, cs := range e.cases {
				acc = FoldExpression(fe, ft, fp, acc, cs.body)
				acc = FoldPattern(ft, fp, acc, cs.pattern)
			}
		}
	case *Tuple:
		{
			e := expr.(*Tuple)
			for _, a := range e.items {
				acc = FoldExpression(fe, ft, fp, acc, a)
			}
		}
	case *Update:
		{
			e := expr.(*Update)
			for _, f := range e.fields {
				acc = FoldExpression(fe, ft, fp, acc, f.value)
			}
		}
	case *Lambda:
		{
			e := expr.(*Lambda)
			for _, p := range e.params {
				acc = FoldPattern(ft, fp, acc, p)
			}
			acc = FoldExpression(fe, ft, fp, acc, e.body)
			acc = FoldType(ft, acc, e.return_)
		}
	case *Accessor:
		{

		}
	case *BinOp:
		{
			e := expr.(*BinOp)
			for _, i := range e.items {
				acc = FoldExpression(fe, ft, fp, acc, i.operand)
			}
		}
	case *Negate:
		{
			e := expr.(*Negate)
			acc = FoldExpression(fe, ft, fp, acc, e.nested)
		}
	case *Constructor:
		{
			e := expr.(*Constructor)
			for _, i := range e.args {
				acc = FoldExpression(fe, ft, fp, acc, i)
			}
		}
	case *InfixVar:
		{
		}
	case *Call:
		{
			e := expr.(*Call)
			for _, i := range e.args {
				acc = FoldExpression(fe, ft, fp, acc, i)
			}
		}
	}
	return acc
}

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
		acc = FoldExpression(fe, ft, fp, acc, &Var{
			ExpressionBase: &ExpressionBase{
				Location: infix.aliasLocation,
			},
			Name: ast.QualifiedIdentifier(infix.alias),
		})
	}
	return acc
}

func FoldDefinition[T any](
	fe func(Expression, T) T, ft func(Type, T) T, fp func(Pattern, T) T, acc T, def *Definition,
) T {
	if def == nil {
		return acc
	}

	for _, p := range def.Params {
		acc = FoldPattern(ft, fp, acc, p)
	}
	acc = FoldType(ft, acc, def.Type)
	acc = FoldExpression(fe, ft, fp, acc, def.Expression)
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
			acc = FoldType(ft, acc, p.type_)
		}
	case *PAny:
		{
			p := pattern.(*PAny)
			acc = FoldType(ft, acc, p.type_)
		}
	case *PCons:
		{
			p := pattern.(*PCons)
			acc = FoldPattern(ft, fp, acc, p.head)
			acc = FoldPattern(ft, fp, acc, p.tail)
			acc = FoldType(ft, acc, p.type_)
		}
	case *PConst:
		{
			p := pattern.(*PConst)
			acc = FoldType(ft, acc, p.type_)
		}
	case *PDataOption:
		{
			p := pattern.(*PDataOption)
			for _, a := range p.values {
				acc = FoldPattern(ft, fp, acc, a)
			}
			acc = FoldType(ft, acc, p.type_)
		}
	case *PList:
		{
			p := pattern.(*PList)
			for _, a := range p.items {
				acc = FoldPattern(ft, fp, acc, a)
			}
			acc = FoldType(ft, acc, p.type_)
		}
	case *PNamed:
		{
			p := pattern.(*PNamed)
			acc = FoldType(ft, acc, p.type_)
		}
	case *PRecord:
		{
			p := pattern.(*PRecord)
			acc = FoldType(ft, acc, p.type_)
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
			acc = FoldExpression(fe, ft, fp, acc, e.Record)
		}
	case *Apply:
		{
			e := expr.(*Apply)
			acc = FoldExpression(fe, ft, fp, acc, e.Func)
			for _, a := range e.Args {
				acc = FoldExpression(fe, ft, fp, acc, a)
			}
		}
	case *Const:
		{

		}
	case *If:
		{
			e := expr.(*If)
			acc = FoldExpression(fe, ft, fp, acc, e.Condition)
			acc = FoldExpression(fe, ft, fp, acc, e.Positive)
			acc = FoldExpression(fe, ft, fp, acc, e.Negative)
		}
	case *LetMatch:
		{
			e := expr.(*LetMatch)
			acc = FoldExpression(fe, ft, fp, acc, e.Value)
			acc = FoldExpression(fe, ft, fp, acc, e.Nested)
			acc = FoldPattern(ft, fp, acc, e.Pattern)
		}
	case *LetDef:
		{
			e := expr.(*LetDef)
			for _, p := range e.Params {
				acc = FoldPattern(ft, fp, acc, p)
			}
			acc = FoldExpression(fe, ft, fp, acc, e.Body)
			acc = FoldType(ft, acc, e.FnType)
			acc = FoldExpression(fe, ft, fp, acc, e.Nested)
		}
	case *List:
		{
			e := expr.(*List)
			for _, a := range e.Items {
				acc = FoldExpression(fe, ft, fp, acc, a)
			}
		}
	case *Record:
		{
			e := expr.(*Record)
			for _, a := range e.Fields {
				acc = FoldExpression(fe, ft, fp, acc, a.Value)
			}
		}
	case *Select:
		{
			e := expr.(*Select)
			acc = FoldExpression(fe, ft, fp, acc, e.Condition)
			for _, cs := range e.Cases {
				acc = FoldExpression(fe, ft, fp, acc, cs.Expression)
				acc = FoldPattern(ft, fp, acc, cs.Pattern)
			}
		}
	case *Tuple:
		{
			e := expr.(*Tuple)
			for _, a := range e.Items {
				acc = FoldExpression(fe, ft, fp, acc, a)
			}
		}
	case *Update:
		{
			e := expr.(*Update)
			for _, f := range e.Fields {
				acc = FoldExpression(fe, ft, fp, acc, f.Value)
			}
		}
	case *Lambda:
		{
			e := expr.(*Lambda)
			for _, p := range e.Params {
				acc = FoldPattern(ft, fp, acc, p)
			}
			acc = FoldExpression(fe, ft, fp, acc, e.Body)
			acc = FoldType(ft, acc, e.Return)
		}
	case *Accessor:
		{

		}
	case *BinOp:
		{
			e := expr.(*BinOp)
			for _, i := range e.Items {
				acc = FoldExpression(fe, ft, fp, acc, i.Expression)
			}
		}
	case *Negate:
		{
			e := expr.(*Negate)
			acc = FoldExpression(fe, ft, fp, acc, e.Nested)
		}
	case *Constructor:
		{
			e := expr.(*Constructor)
			for _, i := range e.Args {
				acc = FoldExpression(fe, ft, fp, acc, i)
			}
		}
	case *InfixVar:
		{
		}
	case *NativeCall:
		{
			e := expr.(*NativeCall)
			for _, i := range e.Args {
				acc = FoldExpression(fe, ft, fp, acc, i)
			}
		}
	}
	return acc
}

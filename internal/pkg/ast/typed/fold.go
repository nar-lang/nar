package typed

func FoldModule[T any](
	fe func(Expression, T) T, ft func(Type, T) T, fp func(Pattern, T) T, acc T, m *Module,
) T {
	for _, def := range m.Definitions {
		acc = FoldDefinition(fe, ft, fp, acc, def)
	}
	return acc
}

func FoldDefinition[T any](
	fe func(Expression, T) T, ft func(Type, T) T, fp func(Pattern, T) T, acc T, def *Definition,
) T {
	acc = FoldType(ft, acc, def.DeclaredType)
	for _, p := range def.Params {
		acc = FoldPattern(ft, fp, acc, p)
	}
	acc = FoldExpression(fe, ft, fp, acc, def.Expression)
	return acc
}

func FoldExpression[T any](
	fe func(Expression, T) T, ft func(Type, T) T, fp func(Pattern, T) T, acc T, expr Expression,
) T {
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
			for _, arg := range e.Args {
				acc = FoldExpression(fe, ft, fp, acc, arg)
			}
		}
	case *Const:
		{
		}
	case *Let:
		{
			e := expr.(*Let)
			acc = FoldExpression(fe, ft, fp, acc, e.Value)
			acc = FoldExpression(fe, ft, fp, acc, e.Body)
			acc = FoldPattern(ft, fp, acc, e.Pattern)
		}
	case *List:
		{
			e := expr.(*List)
			for _, item := range e.Items {
				acc = FoldExpression(fe, ft, fp, acc, item)
			}
		}
	case *Record:
		{
			e := expr.(*Record)
			for _, item := range e.Fields {
				acc = FoldExpression(fe, ft, fp, acc, item.Value)
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
			for _, item := range e.Items {
				acc = FoldExpression(fe, ft, fp, acc, item)
			}
		}
	case *UpdateLocal:
		{
			e := expr.(*UpdateLocal)
			for _, fl := range e.Fields {
				acc = FoldExpression(fe, ft, fp, acc, fl.Value)
			}
		}
	case *UpdateGlobal:
		{
			e := expr.(*UpdateGlobal)
			for _, fl := range e.Fields {
				acc = FoldExpression(fe, ft, fp, acc, fl.Value)
			}
		}
	case *Constructor:
		{
			e := expr.(*Constructor)
			for _, arg := range e.Args {
				acc = FoldExpression(fe, ft, fp, acc, arg)
			}
		}
	case *NativeCall:
		{
			e := expr.(*NativeCall)
			for _, arg := range e.Args {
				acc = FoldExpression(fe, ft, fp, acc, arg)
			}
		}
	case *Local:
		{

		}
	case *Global:
		{

		}
	default:
		panic("unreachable")
	}
	return acc
}

func FoldPattern[T any](ft func(Type, T) T, fp func(Pattern, T) T, acc T, pt Pattern) T {
	acc = fp(pt, acc)
	acc = FoldType(ft, acc, pt.GetDeclaredType())
	switch pt.(type) {
	case *PAlias:
		{
			e := pt.(*PAlias)
			acc = FoldPattern(ft, fp, acc, e.Nested)
		}
	case *PAny:
		{
		}
	case *PCons:
		{
			e := pt.(*PCons)
			acc = FoldPattern(ft, fp, acc, e.Head)
			acc = FoldPattern(ft, fp, acc, e.Tail)
		}
	case *PConst:
		{

		}
	case *PDataOption:
		{
			e := pt.(*PDataOption)
			for _, arg := range e.Args {
				acc = FoldPattern(ft, fp, acc, arg)
			}
		}
	case *PList:
		{
			e := pt.(*PList)
			for _, arg := range e.Items {
				acc = FoldPattern(ft, fp, acc, arg)
			}
		}
	case *PNamed:
		{
		}
	case *PRecord:
		{
		}
	case *PTuple:
		{
			e := pt.(*PTuple)
			for _, arg := range e.Items {
				acc = FoldPattern(ft, fp, acc, arg)
			}
		}
	default:
		panic("unreachable")
	}
	return acc
}

func FoldType[T any](ft func(Type, T) T, acc T, t Type) T {
	if t == nil {
		return acc
	}
	acc = ft(t, acc)
	switch t.(type) {
	case *TFunc:
		{
			e := t.(*TFunc)
			for _, arg := range e.Params {
				acc = FoldType(ft, acc, arg)
			}
			acc = FoldType(ft, acc, e.Return)
		}
	case *TRecord:
		{
			e := t.(*TRecord)
			for _, f := range e.Fields {
				acc = FoldType(ft, acc, f)
			}
		}
	case *TTuple:
		{
			e := t.(*TTuple)
			for _, arg := range e.Items {
				acc = FoldType(ft, acc, arg)
			}
		}
	case *TNative:
		{
			e := t.(*TNative)
			for _, arg := range e.Args {
				acc = FoldType(ft, acc, arg)
			}
		}
	case *TData:
		{
			e := t.(*TData)
			for _, arg := range e.Args {
				acc = FoldType(ft, acc, arg)
			}
			/*for _, do := range e.Options {
				for _, v := range do.Values {
					acc = FoldType(ft, acc, v)
				}
			}*/
		}
	case *TUnbound:
		{

		}
	default:
		panic("unreachable")
	}
	return acc
}

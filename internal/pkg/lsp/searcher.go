package lsp

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/parsed"
)

func findStatement(
	loc ast.Location, module *parsed.Module,
) (*parsed.Definition, parsed.Expression, parsed.Type) {
	//var findType func(loc ast.Location, t parsed.Type) parsed.Type
	var findExpression func(loc ast.Location, expr parsed.Expression) (parsed.Expression, parsed.Type)

	/*findType = func(
		loc ast.Location, t parsed.Type,
	) parsed.Type {
		switch t.(type) {
		case parsed.TRecord:
			e := t.(parsed.TRecord)
			for _, f := range e.Fields {
				if f.GetLocation().Contains(loc) {
					return findType(loc, f)
				}
			}
			break
		case parsed.TTuple:
			e := t.(parsed.TTuple)
			for _, f := range e.Items {
				if f.GetLocation().Contains(loc) {
					return findType(loc, f)
				}
			}
			break
		case parsed.TFunc:
			e := t.(parsed.TFunc)
			for _, f := range e.Params {
				if f.GetLocation().Contains(loc) {
					return findType(loc, f)
				}
			}
			if e.Return.GetLocation().Contains(loc) {
				return findType(loc, e.Return)
			}
			break
		case parsed.TData:
			e := t.(parsed.TData)
			for _, f := range e.Options {
				for _, v := range f.Values {
					if v.GetLocation().Contains(loc) {
						return findType(loc, v)
					}
				}
			}
			break
		}
		return t
	}*/

	findExpression = func(
		loc ast.Location, expr parsed.Expression,
	) (parsed.Expression, parsed.Type) {
		if expr == nil {
			return nil, nil
		}

		switch expr.(type) {
		case parsed.Access:
			e := expr.(parsed.Access)
			if e.Record != nil && e.Record.GetLocation().Contains(loc) {
				return findExpression(loc, e.Record)
			}
			break
		case parsed.Apply:
			e := expr.(parsed.Apply)
			for _, a := range e.Args {
				if a != nil && a.GetLocation().Contains(loc) {
					return findExpression(loc, a)
				}
			}
			if e.Func != nil && e.Func.GetLocation().Contains(loc) {
				return findExpression(loc, e.Func)
			}
			break
		case parsed.Const:
			break
		case parsed.If:
			e := expr.(parsed.If)
			if e.Condition != nil && e.Condition.GetLocation().Contains(loc) {
				return findExpression(loc, e.Condition)
			}
			if e.Positive != nil && e.Positive.GetLocation().Contains(loc) {
				return findExpression(loc, e.Positive)
			}
			if e.Negative != nil && e.Negative.GetLocation().Contains(loc) {
				return findExpression(loc, e.Negative)
			}
			break
		case parsed.LetMatch:
			e := expr.(parsed.LetMatch)
			if e.Nested != nil && e.Nested.GetLocation().Contains(loc) {
				return findExpression(loc, e.Nested)
			}
			if e.Value != nil && e.Value.GetLocation().Contains(loc) {
				return findExpression(loc, e.Value)
			}
			break
		case parsed.LetDef:
			e := expr.(parsed.LetDef)
			if e.Body != nil && e.Body.GetLocation().Contains(loc) {
				return findExpression(loc, e.Nested)
			}
			if e.Body != nil && e.Nested.GetLocation().Contains(loc) {
				return findExpression(loc, e.Nested)
			}
			break
		case parsed.List:
			e := expr.(parsed.List)
			for _, a := range e.Items {
				if a != nil && a.GetLocation().Contains(loc) {
					return findExpression(loc, a)
				}
			}
			break
		case parsed.Record:
			e := expr.(parsed.Record)
			for _, f := range e.Fields {
				if f.Value != nil && f.Value.GetLocation().Contains(loc) {
					return findExpression(loc, f.Value)
				}
			}
			break
		case parsed.Select:
			e := expr.(parsed.Select)
			if e.Condition != nil && e.Condition.GetLocation().Contains(loc) {
				return findExpression(loc, e.Condition)
			}
			for _, cs := range e.Cases {
				if cs.Expression != nil && cs.Expression.GetLocation().Contains(loc) {
					return findExpression(loc, cs.Expression)
				}
			}
		case parsed.Tuple:
			e := expr.(parsed.Tuple)
			for _, f := range e.Items {
				if f != nil && f.GetLocation().Contains(loc) {
					return findExpression(loc, f)
				}
			}
			break
		case parsed.Update:
			e := expr.(parsed.Update)
			for _, f := range e.Fields {
				if f.Value != nil && f.Value.GetLocation().Contains(loc) {
					return findExpression(loc, f.Value)
				}
			}
			break
		case parsed.Lambda:
			e := expr.(parsed.Lambda)
			if e.Body != nil && e.Body.GetLocation().Contains(loc) {
				return findExpression(loc, e.Body)
			}
			break
		case parsed.Accessor:
			break
		case parsed.BinOp:
			e := expr.(parsed.BinOp)
			for _, i := range e.Items {
				if i.Expression != nil && i.Expression.GetLocation().Contains(loc) {
					return findExpression(loc, i.Expression)
				}
			}
			break
		case parsed.Negate:
			e := expr.(parsed.Negate)
			if e.Nested != nil && e.Nested.GetLocation().Contains(loc) {
				return findExpression(loc, e.Nested)
			}
			break
		case parsed.Constructor:
			e := expr.(parsed.Constructor)
			for _, a := range e.Args {
				if a != nil && a.GetLocation().Contains(loc) {
					return findExpression(loc, a)
				}
			}
			break
		case parsed.InfixVar:
			break
		case parsed.NativeCall:
			e := expr.(parsed.NativeCall)
			for _, a := range e.Args {
				if a != nil && a.GetLocation().Contains(loc) {
					return findExpression(loc, a)
				}
			}
			break
		}

		return expr, nil
	}

	findDefinition := func(
		loc ast.Location, def parsed.Definition,
	) (*parsed.Definition, parsed.Expression, parsed.Type) {
		if def.Expression != nil && def.Expression.GetLocation().Contains(loc) {
			e, t := findExpression(loc, def.Expression)
			return &def, e, t
		}
		return &def, nil, nil
	}

	for _, def := range module.Definitions {
		if def.Location.Contains(loc) {
			return findDefinition(loc, def)
		}
	}

	return nil, nil, nil
}

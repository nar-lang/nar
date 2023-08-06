package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewChainExpression(c misc.Cursor, args []Expression) Expression {
	return expressionChain{cursor: c, Args: args}
}

type expressionChain struct {
	ExpressionChain__ int
	Args              []Expression
	cursor            misc.Cursor
}

func (e expressionChain) getCursor() misc.Cursor {
	return e.cursor
}

func (e expressionChain) precondition(md *Metadata) (Expression, error) {
	args, err := infixToPrefix(e.Args, md)
	if err != nil {
		return nil, err
	}

	ex, err := unwrapChain(expressionChain{Args: args}, md)
	if err != nil {
		return nil, err
	}
	return ex.precondition(md)
}

func (e expressionChain) setType(type_ Type, gm genericsMap, md *Metadata) (Expression, Type, error) {
	return nil, nil, misc.NewError(e.cursor, "trying to set type to chain expression (this is a compiler error)")
}

func (e expressionChain) getType(md *Metadata) (Type, error) {
	return nil, misc.NewError(e.cursor, "trying to get type to chain expression (this is a compiler error)")
}

func (e expressionChain) resolve(md *Metadata) (resolved.Expression, error) {
	return nil, misc.NewError(e.cursor, "trying to resolve an expression chain (this is a compiler error)")
}

func unwrapChain(chain Expression, md *Metadata) (Expression, error) {
	if exChain, ok := chain.(expressionChain); ok {
		exs := exChain.Args

		if len(exs) == 1 {
			if e, ok := exs[0].(expressionChain); ok {
				return unwrapChain(e, md)
			}
			if _, ok := exs[0].(expressionIdentifier); !ok {
				return exs[0], nil
			}
		}

		ident, ok := exs[0].(expressionIdentifier)

		if !ok {
			infix, ok := exs[0].(ExpressionInfix)
			if !ok {
				return nil, misc.NewError(exs[0].getCursor(), "expected function here")
			}
			ident = expressionIdentifier{Name: infix.name, cursor: exs[0].getCursor()}
		}

		type_, _, err := md.getTypeByName(md.currentModuleName(), ident.Name, ident.GenericArgs, ident.cursor)
		if err != nil {
			return nil, err
		}

		if len(exs) == 1 {
			if ts, ok := type_.(typeSignature); ok {
				generics := ident.GenericArgs
				if len(generics) == 0 {
					generics = ts.ReturnType.getGenerics()
				}
				return expressionApply{
					Name:        ident.Name,
					Args:        []Expression{NewConstExpression(ident.cursor, ConstKindVoid, "")},
					GenericArgs: generics,
					cursor:      ident.cursor,
				}, nil
			} else {
				return expressionValue{Name: ident.Name, cursor: ident.cursor}, nil
			}
		} else {
			var args []Expression
			for _, ex := range exs[1:] {
				unwrappedChain, err := unwrapChain(expressionChain{Args: []Expression{ex}}, md)
				if err != nil {
					return nil, err
				}
				args = append(args, unwrappedChain)
			}
			return expressionApply{
				Name:        ident.Name,
				Args:        args,
				GenericArgs: ident.GenericArgs,
				cursor:      ident.cursor,
			}, nil
		}
	}
	return chain, nil
}

func infixToPrefix(exs []Expression, md *Metadata) ([]Expression, error) {
	index := -1
	assocCollision := -1
	var maxPriority int
	var currentAssoc InfixAssociativity
	var neg = false

	for i, e := range exs {
		if infixExpr, ok := e.(ExpressionInfix); ok && !infixExpr.asParameter {
			if infixExpr.isNegateOp() {
				index = i
				neg = true
				assocCollision = -1
				break
			}

			type_, _, err := md.getTypeByName(md.currentModuleName(), infixExpr.name, nil, infixExpr.cursor)
			if err != nil {
				return nil, err
			}

			infix, ok := type_.(typeInfix)
			if !ok {
				return nil, misc.NewError(e.getCursor(), "expected infix function here")
			}

			if infix.definition.Priority < maxPriority || index < 0 {
				maxPriority = infix.definition.Priority
				index = i
				currentAssoc = infix.definition.Associativity
				assocCollision = -1
			} else if infix.definition.Priority == maxPriority {
				if currentAssoc != infix.definition.Associativity || currentAssoc == InfixAssociativityNon {
					if assocCollision < 0 {
						assocCollision = index
					}
				}
				if currentAssoc == InfixAssociativityRight {
					index = i
				}
			}
		}
	}
	if index >= 0 {
		if neg {
			if len(exs) == index+1 {
				return nil, misc.NewError(exs[index].getCursor(), "unary minus requires operand")
			}
			return infixToPrefix(
				append(
					append(
						exs[0:index],
						expressionChain{
							Args: Expressions{
								expressionIdentifier{
									cursor: exs[index].getCursor(),
									Name:   "neg",
								},
								exs[index+1],
							},
							cursor: exs[index].getCursor(),
						},
					),
					exs[index+2:]...,
				), md)
		} else {
			if assocCollision >= 0 {
				return nil, misc.NewError(
					exs[index].getCursor(),
					"cannot resolve infix priority of `%s` and `%s`, use brackets",
					exs[index].(ExpressionInfix).name,
					exs[assocCollision].(ExpressionInfix).name,
				)
			}
			if index == 0 {
				if len(exs) != 3 {
					return nil, misc.NewError(exs[index].getCursor(), "infix function expects 2 arguments")
				}
			} else {
				left, err := infixToPrefix(exs[0:index], md)
				if err != nil {
					return nil, err
				}
				right, err := infixToPrefix(exs[index+1:], md)
				if err != nil {
					return nil, err
				}

				exs = []Expression{exs[index], expressionChain{Args: left}, expressionChain{Args: right}}
			}
		}
	} else {
		if len(exs) == 1 {
			if c, ok := exs[0].(expressionChain); ok {
				var err error
				exs, err = infixToPrefix(c.Args, md)
				if err != nil {
					return nil, err
				}
			}
		} else {
			for i, ex := range exs {
				if ec, ok := ex.(expressionChain); ok {
					args, err := infixToPrefix(ec.Args, md)
					if err != nil {
						return nil, err
					}
					exs[i] = expressionChain{Args: args}
				}

			}
		}
	}
	return exs, nil
}

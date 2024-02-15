package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

func NewBinOp(location ast.Location, items []*BinOpItem, inParentheses bool) Expression {
	return &BinOp{
		expressionBase: newExpressionBase(location),
		items:          items,
		inParentheses:  inParentheses,
	}
}

type BinOp struct {
	*expressionBase
	items         []*BinOpItem
	inParentheses bool
}

func (e *BinOp) Iterate(f func(statement Statement)) {
	f(e)
	for _, i := range e.items {
		if i.operand != nil {
			i.operand.Iterate(f)
		}
	}
}

func (e *BinOp) SetInParentheses(inParentheses bool) {
	e.inParentheses = inParentheses
}

func (e *BinOp) InParentheses() bool {
	return e.inParentheses
}

func (e *BinOp) Items() []*BinOpItem {
	return e.items
}

func (e *BinOp) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Expression, error) {
	var output []*BinOpItem
	var operators []*BinOpItem
	for _, o1 := range e.items {
		if o1.operand != nil {
			output = append(output, o1)
		} else {
			if infixFn, _, ids := module.findInfixFn(modules, o1.infix); len(ids) != 1 {
				return nil, newAmbiguousInfixError(ids, o1.infix, e.location)
			} else {
				o1.fn = infixFn
			}

			for i := len(operators) - 1; i >= 0; i-- {
				o2 := operators[i]
				if o1.fn.hasLowerPrecedenceThan(o2.fn) {
					output = append(output, o2)
					operators = operators[:len(operators)-1]
				} else {
					break
				}
			}
			operators = append(operators, o1)
		}
	}
	for i := len(operators) - 1; i >= 0; i-- {
		output = append(output, operators[i])
	}

	var buildTree func() (normalized.Expression, error)
	buildTree = func() (normalized.Expression, error) {
		op := output[len(output)-1].infix
		output = output[:len(output)-1]

		if infixA, m, ids := module.findInfixFn(modules, op); len(ids) != 1 {
			return nil, newAmbiguousInfixError(ids, op, e.location)
		} else {
			var left, right normalized.Expression
			var err error
			r := output[len(output)-1]
			if r.operand != nil {
				right, err = r.operand.normalize(locals, modules, module, normalizedModule)
				if err != nil {
					return nil, err
				}
				output = output[:len(output)-1]
			} else {
				right, err = buildTree()
				if err != nil {
					return nil, err
				}
			}

			l := output[len(output)-1]
			if l.operand != nil {
				left, err = l.operand.normalize(locals, modules, module, normalizedModule)
				if err != nil {
					return nil, err
				}
				output = output[:len(output)-1]
			} else {
				left, err = buildTree()
				if err != nil {
					return nil, err
				}
			}

			return normalized.NewApply(
				e.location,
				normalized.NewGlobal(e.location, m.name, infixA.alias()),
				[]normalized.Expression{left, right},
			), nil
		}
	}

	tree, err := buildTree()
	if err != nil {
		return nil, err
	}
	return e.setSuccessor(tree)
}

type BinOpItem struct {
	operand Expression
	infix   ast.InfixIdentifier
	fn      Infix
}

func NewBinOpOperand(expression Expression) *BinOpItem {
	return &BinOpItem{
		operand: expression,
	}
}

func NewBinOpFunc(infix ast.InfixIdentifier) *BinOpItem {
	return &BinOpItem{
		infix: infix,
	}
}

package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewSelectExpression(c misc.Cursor, condition Expression, cases []ExpressionSelectCase) Expression {
	return expressionSelect{cursor: c, Condition: condition, Cases: cases}
}

type expressionSelect struct {
	ExpressionSelect__ int
	Condition          Expression
	Cases              []ExpressionSelectCase
	cursor             misc.Cursor
}

func (e expressionSelect) getCursor() misc.Cursor {
	return e.cursor
}

func (e expressionSelect) precondition(md *Metadata) (Expression, error) {
	var err error
	e.Condition, err = e.Condition.precondition(md)
	if err != nil {
		return nil, err
	}
	for i, cs := range e.Cases {
		locals := md.cloneLocalVars()
		t, err := e.Condition.getType(md)
		if err != nil {
			return nil, err
		}
		err = cs.Decons.extractLocals(t, md)
		if err != nil {
			return nil, err
		}
		cs.Expression, err = cs.Expression.precondition(md)
		if err != nil {
			return nil, err
		}
		md.LocalVars = locals
		e.Cases[i] = cs
	}
	return e, nil
}

func (e expressionSelect) setType(type_ Type, gm genericsMap, md *Metadata) (Expression, Type, error) {
	t, err := e.Condition.getType(md)
	if err != nil {
		return nil, nil, err
	}
	e.Condition, t, err = e.Condition.setType(t, gm, md)
	if err != nil {
		return nil, nil, err
	}
	var exprType Type
	for i, cs := range e.Cases {
		locals := md.cloneLocalVars()
		err = cs.Decons.extractLocals(t, md)
		if err != nil {
			return nil, nil, err
		}
		var inferredType Type
		cs.Expression, inferredType, err = cs.Expression.setType(type_, gm, md)
		if err != nil {
			return nil, nil, err
		}
		if i == 0 {
			exprType, err = inferredType.dereference(md)
		} else {
			if !typesEqual(exprType, inferredType, false, md) {
				return nil, nil, misc.NewError(
					cs.Expression.getCursor(),
					"case expression types do not match, expected %s got %s",
					exprType,
					inferredType,
				)
			}
		}
		if err != nil {
			return nil, nil, err
		}
		e.Cases[i] = cs
		md.LocalVars = locals
	}

	return e, exprType, nil
}

func (e expressionSelect) getType(md *Metadata) (Type, error) {
	locals := md.cloneLocalVars()
	t, err := e.Condition.getType(md)
	if err != nil {
		return nil, err
	}
	_, err = e.Cases[0].Decons.resolve(t, md)
	if err != nil {
		return nil, err
	}
	_type, _ := e.Cases[0].Expression.getType(md)
	md.LocalVars = locals
	return _type, nil
}

func (e expressionSelect) resolve(md *Metadata) (resolved.Expression, error) {
	resolvedCondition, err := e.Condition.resolve(md)
	if err != nil {
		return nil, err
	}

	var resolvedCases []resolved.ExpressionSelectCase
	for _, cs := range e.Cases {
		locals := md.cloneLocalVars()
		t, err := e.Condition.getType(md)
		if err != nil {
			return nil, err
		}
		resolvedDecons, err := cs.Decons.resolve(t, md)
		if err != nil {
			return nil, err
		}
		resolvedExpression, err := cs.Expression.resolve(md)
		resolvedCases = append(resolvedCases, resolved.NewExpressionSelectCase(resolvedDecons, resolvedExpression))
		md.LocalVars = locals
	}
	return resolved.NewSelectExpression(resolvedCondition, resolvedCases), nil
}

type ExpressionSelectCase struct {
	Decons     Decons
	Expression Expression
}

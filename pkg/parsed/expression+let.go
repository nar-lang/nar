package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewLetExpression(c misc.Cursor, definitions []LetDefinition, expression Expression) Expression {
	return expressionLet{cursor: c, Definitions: definitions, Expression: expression}
}

type expressionLet struct {
	Definitions []LetDefinition
	Expression  Expression
	cursor      misc.Cursor
}

func (e expressionLet) getCursor() misc.Cursor {
	return e.cursor
}

func (e expressionLet) precondition(md *Metadata) (Expression, error) {
	var err error
	locals := md.cloneLocalVars()
	for i, def := range e.Definitions {
		err = def.param.extractLocals(def.type_, md)
		if err != nil {
			return nil, err
		}
		innerLocals := md.cloneLocalVars()
		err = def.type_.extractLocals(def.type_, md)
		if err != nil {
			return nil, err
		}
		def.expression, err = def.expression.precondition(md)
		if err != nil {
			return nil, err
		}
		e.Definitions[i] = def
		md.LocalVars = innerLocals
	}
	e.Expression, err = e.Expression.precondition(md)
	if err != nil {
		return nil, err
	}
	md.LocalVars = locals
	return e, nil
}

func (e expressionLet) setType(type_ Type, md *Metadata) (Expression, Type, error) {
	var err error
	locals := md.cloneLocalVars()
	for i, def := range e.Definitions {

		innerLocals := md.cloneLocalVars()
		err = def.type_.extractLocals(def.type_, md)
		if err != nil {
			return nil, nil, err
		}
		exprType := def.type_

		dt, err := def.type_.dereference(md)
		if err != nil {
			return nil, nil, err
		}
		signature, ok := dt.(typeSignature)
		if ok {
			_, exprType = signature.flattenDefinition()
		}

		def.expression, _, err = def.expression.setType(exprType, md)
		if err != nil {
			return nil, nil, err
		}
		e.Definitions[i] = def
		md.LocalVars = innerLocals

		err = def.param.extractLocals(def.type_, md)
		if err != nil {
			return nil, nil, err
		}
	}
	var inferredType Type
	e.Expression, inferredType, err = e.Expression.setType(type_, md)
	if err != nil {
		return nil, nil, err
	}
	md.LocalVars = locals
	return e, inferredType, nil
}

func (e expressionLet) getType(md *Metadata) (Type, error) {
	return e.Expression.getType(md)
}

func (e expressionLet) resolve(md *Metadata) (resolved.Expression, error) {
	locals := md.cloneLocalVars()
	var resolvedLets []resolved.LetDefinition
	for _, def := range e.Definitions {
		innerLocals := md.cloneLocalVars()
		err := def.type_.extractLocals(def.type_, md)
		if err != nil {
			return nil, err
		}
		resolvedExpression, err := def.expression.resolve(md)
		if err != nil {
			return nil, err
		}
		resolvedParam, err := def.param.resolve(def.type_, md)
		if err != nil {
			return nil, err
		}
		resolvedType, err := def.type_.resolve(e.cursor, md)
		if err != nil {
			return nil, err
		}
		resolvedLets = append(resolvedLets, resolved.NewLetDefinition(resolvedParam, resolvedType, resolvedExpression))
		md.LocalVars = innerLocals
		err = def.param.extractLocals(def.type_, md)
		if err != nil {
			return nil, err
		}
	}
	resolvedExpression, err := e.Expression.resolve(md)
	if err != nil {
		return nil, err
	}
	result := resolved.NewLetExpression(resolvedLets, resolvedExpression)
	md.LocalVars = locals
	return result, nil
}

func NewLetDefinition(param Parameter, type_ Type, expression Expression) LetDefinition {
	return LetDefinition{param: param, type_: type_, expression: expression}
}

type LetDefinition struct {
	param      Parameter
	type_      Type
	expression Expression
}

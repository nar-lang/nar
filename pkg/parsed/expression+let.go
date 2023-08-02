package parsed

import (
	"encoding/json"
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewLetExpression(c misc.Cursor, definitions []LetDefinition, expression Expression) Expression {
	return expressionLet{cursor: c, Definitions: definitions, Expression: expression}
}

type expressionLet struct {
	ExpressionLet__ int
	Definitions     []LetDefinition
	Expression      Expression
	cursor          misc.Cursor
}

func (e expressionLet) getCursor() misc.Cursor {
	return e.cursor
}

func (e expressionLet) precondition(md *Metadata) (Expression, error) {
	var err error
	locals := md.cloneLocalVars()
	for i, def := range e.Definitions {
		md.LocalVars[def.name] = def.type_
		def.expression, err = def.expression.precondition(md)
		if err != nil {
			return nil, err
		}
		e.Definitions[i] = def
	}
	e.Expression, err = e.Expression.precondition(md)
	if err != nil {
		return nil, err
	}
	md.LocalVars = locals
	return e, nil
}

func (e expressionLet) setType(type_ Type, gm genericsMap, md *Metadata) (Expression, Type, error) {
	var err error
	locals := md.cloneLocalVars()
	for i, def := range e.Definitions {
		def.expression, def.type_, err = def.expression.setType(def.type_, gm, md)
		if err != nil {
			return nil, nil, err
		}
		e.Definitions[i] = def
		md.LocalVars[def.name] = def.type_
	}
	var inferredType Type
	e.Expression, inferredType, err = e.Expression.setType(type_, gm, md)
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
	for _, definition := range e.Definitions {
		resolvedExpression, err := definition.expression.resolve(md)
		if err != nil {
			return nil, err
		}
		resolvedLets = append(resolvedLets, resolved.NewLetDefinition(definition.name, resolvedExpression))
		md.LocalVars[definition.name] = definition.type_
	}
	resolvedExpression, err := e.Expression.resolve(md)
	if err != nil {
		return nil, err
	}
	result := resolved.NewLetExpression(resolvedLets, resolvedExpression)
	md.LocalVars = locals
	return result, nil
}

func NewLetDefinition(name string, type_ Type, expression Expression) LetDefinition {
	return LetDefinition{name: name, type_: type_, expression: expression}
}

type LetDefinition struct {
	name       string
	type_      Type
	expression Expression
}

func (d LetDefinition) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name       string
		Type       Type
		Expression Expression
	}{
		Name:       d.name,
		Type:       d.type_,
		Expression: d.expression,
	})
}

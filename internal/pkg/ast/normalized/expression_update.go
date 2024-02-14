package normalized

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
)

type Update struct {
	*expressionBase
	moduleName ast.QualifiedIdentifier
	recordName ast.Identifier
	fields     []*RecordField
	target     Pattern
}

func NewUpdateLocal(loc ast.Location, recordName ast.Identifier, target Pattern, fields []*RecordField) Expression {
	return &Update{
		expressionBase: newExpressionBase(loc),
		recordName:     recordName,
		fields:         fields,
		target:         target,
	}
}

func (e *Update) flattenLambdas(parentName ast.Identifier, m *Module, locals map[ast.Identifier]Pattern) Expression {
	for i, a := range e.fields {
		e.fields[i].value = a.value.flattenLambdas(parentName, m, locals)
	}
	return e
}

func (e *Update) replaceLocals(replace map[ast.Identifier]Expression) Expression {
	for i, a := range e.fields {
		e.fields[i].value = a.value.replaceLocals(replace)
	}
	return e
}

func (e *Update) extractUsedLocalsSet(definedLocals map[ast.Identifier]Pattern, usedLocals map[ast.Identifier]struct{}) {
	for _, f := range e.fields {
		f.value.extractUsedLocalsSet(definedLocals, usedLocals)
	}
}

func NewUpdateGlobal(
	loc ast.Location,
	moduleName ast.QualifiedIdentifier,
	definitionName ast.Identifier,
	fields []*RecordField,
) Expression {
	return &Update{
		expressionBase: newExpressionBase(loc),
		moduleName:     moduleName,
		recordName:     definitionName,
		fields:         fields,
	}
}

func (e *Update) annotate(ctx *typed.SolvingContext, typeParams typeParamsMap, modules map[ast.QualifiedIdentifier]*Module, typedModules map[ast.QualifiedIdentifier]*typed.Module, moduleName ast.QualifiedIdentifier, stack []*typed.Definition) (typed.Expression, error) {
	fields, err := common.MapError(func(f *RecordField) (*typed.RecordField, error) {
		value, err := f.value.annotate(ctx, typeParams, modules, typedModules, moduleName, stack)
		if err != nil {
			return nil, err
		}
		return typed.NewRecordField(ctx, f.location, f.name, value), nil
	}, e.fields)
	if err != nil {
		return nil, err
	}

	if e.moduleName != "" {
		targetDef, err := getAnnotatedGlobal(
			e.moduleName, e.recordName, modules, typedModules, stack, e.location)
		if err != nil {
			return nil, err
		}
		return e.setSuccessor(typed.NewUpdateGlobal(ctx, e.location, e.moduleName, e.recordName, targetDef, fields))
	} else {
		if e.target == nil {
			return nil, common.Error{
				Location: e.location,
				Message:  fmt.Sprintf("local variable `%s` not resolved", e.recordName),
			}
		}
		return e.setSuccessor(typed.NewUpdateLocal(e.location, e.recordName, e.target.Successor(), fields))
	}
}

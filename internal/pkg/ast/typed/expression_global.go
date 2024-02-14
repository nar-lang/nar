package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/bytecode"
	"nar-compiler/internal/pkg/common"
)

type Global struct {
	*expressionBase
	moduleName     ast.QualifiedIdentifier
	definitionName ast.Identifier
	definition     *Definition
}

func NewGlobal(
	ctx *SolvingContext, loc ast.Location,
	moduleName ast.QualifiedIdentifier, definitionName ast.Identifier,
	targetDef *Definition,
) Expression {
	return ctx.annotateExpression(&Global{
		expressionBase: newExpressionBase(loc),
		moduleName:     moduleName,
		definitionName: definitionName,
		definition:     targetDef,
	})
}

func (e *Global) checkPatterns() error {
	return nil
}

func (e *Global) mapTypes(subst map[uint64]Type) error {
	var err error
	e.type_, err = e.type_.mapTo(subst)
	if err != nil {
		return err
	}
	return nil
}

func (e *Global) Code(currentModule ast.QualifiedIdentifier) string {
	name := string(e.definitionName)
	if currentModule != e.moduleName {
		name = string(common.MakeFullIdentifier(e.moduleName, e.definitionName))
	}
	return fmt.Sprintf("%s", name)
}

func (e *Global) appendEquations(eqs Equations, loc *ast.Location, localDefs localTypesMap, ctx *SolvingContext, stack []*Definition) (Equations, error) {
	if e.definition == nil {
		return nil, common.Error{
			Location: e.location,
			Message:  fmt.Sprintf("definition `%s` not found", e.definitionName),
		}
	}

	defType, err := e.definition.uniqueType(ctx, stack)
	if err != nil {
		return nil, err
	}

	eqs = append(eqs, NewEquation(e, e.type_, defType))
	return eqs, nil
}

func (e *Global) appendBytecode(ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary) ([]bytecode.Op, []ast.Location) {
	id := common.MakeFullIdentifier(e.moduleName, e.definitionName)
	funcIndex, ok := binary.FuncsMap[id]
	if !ok {
		panic(common.Error{
			Location: e.location,
			Message:  fmt.Sprintf("global definition `%s` not found", id),
		}.Error())
	}
	ops, locations = bytecode.AppendLoadGlobal(funcIndex, e.location, ops, locations)
	return ops, locations
}

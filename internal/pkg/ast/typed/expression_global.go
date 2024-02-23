package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
	bytecode "nar-compiler/pkg/bytecode"
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
		return nil, common.NewErrorOf(e, "definition `%s` not found", e.definitionName)
	}

	defType, err := e.definition.uniqueType(ctx, stack)
	if err != nil {
		return nil, err
	}

	eqs = append(eqs, NewEquation(e, e.type_, defType))
	return eqs, nil
}

func (e *Global) appendBytecode(ops []bytecode.Op, locations []bytecode.Location, binary *bytecode.Binary) ([]bytecode.Op, []bytecode.Location) {
	id := common.MakeFullIdentifier(e.moduleName, e.definitionName)
	funcIndex, ok := binary.FuncsMap[bytecode.FullIdentifier(id)]
	if !ok {
		panic(common.NewErrorOf(e, "global definition `%s` not found", id).Error())
	}
	ops, locations = bytecode.AppendLoadGlobal(funcIndex, e.location.Bytecode(), ops, locations)
	return ops, locations
}

func (e *Global) Definition() *Definition {
	return e.definition
}

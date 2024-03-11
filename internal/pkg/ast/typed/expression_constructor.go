package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
	"nar-compiler/pkg/bytecode"
)

type Constructor struct {
	*expressionBase
	dataName   ast.FullIdentifier
	optionName ast.Identifier
	dataType   *TData
	args       []Expression
}

func NewConstructor(
	ctx *SolvingContext, loc ast.Location,
	dataName ast.FullIdentifier, optionName ast.Identifier,
	dataType *TData, args []Expression,
) Expression {
	return ctx.annotateExpression(&Constructor{
		expressionBase: newExpressionBase(loc),
		dataName:       dataName,
		optionName:     optionName,
		dataType:       dataType,
		args:           args,
	})
}

func (e *Constructor) checkPatterns() error {
	for _, arg := range e.args {
		if err := arg.checkPatterns(); err != nil {
			return err
		}
	}
	return nil
}

func (e *Constructor) mapTypes(subst map[uint64]Type) error {
	var err error
	e.type_, err = e.type_.mapTo(subst)
	if err != nil {
		return err
	}
	for _, arg := range e.args {
		err = arg.mapTypes(subst)
		if err != nil {
			return err
		}
	}
	if e.dataType != nil {
		if xdt, err := e.dataType.mapTo(subst); err != nil {
			return err
		} else if txdt, ok := xdt.(*TData); !ok {
			return common.NewErrorOf(e.dataType, "failed to map data type")
		} else {
			e.dataType = txdt
		}
	}
	return nil
}

func (e *Constructor) Children() []Statement {
	ch := append(e.expressionBase.Children(), e.dataType)
	return append(ch, common.Map(func(x Expression) Statement { return x }, e.args)...)
}

func (e *Constructor) Code(currentModule ast.QualifiedIdentifier) string {
	args := common.Fold(
		func(x Expression, s string) string {
			if s != "" {
				s += ", "
			}
			return s + x.Code(currentModule)
		}, "", e.args)
	if args != "" {
		args = "(" + args + ")"
	}
	return fmt.Sprintf("%s%s", e.dataName, args)
}

func (e *Constructor) appendEquations(eqs Equations, loc *ast.Location, localDefs localTypesMap, ctx *SolvingContext, stack []*Definition) (Equations, error) {
	var err error
	var r Type
	if e.dataType == nil {
		r = NewTData(e.location, e.dataName, nil, nil)
	} else {
		r = NewTData(e.location, e.dataName, e.dataType.args, e.dataType.options)
	}
	eqs = append(eqs, NewEquation(e, e.type_, r))
	for _, a := range e.args {
		eqs, err = a.appendEquations(eqs, loc, localDefs, ctx, stack)
		if err != nil {
			return nil, err
		}
	}
	return eqs, nil
}

func (e *Constructor) appendBytecode(ops []bytecode.Op, locations []bytecode.Location, binary *bytecode.Binary, hash *bytecode.BinaryHash) ([]bytecode.Op, []bytecode.Location) {
	for _, arg := range e.args {
		ops, locations = arg.appendBytecode(ops, locations, binary, hash)
	}
	ops, locations = ast.CString{
		Value: string(common.MakeDataOptionIdentifier(e.dataName, e.optionName)),
	}.AppendBytecode(bytecode.StackKindObject, e.location, ops, locations, binary, hash)
	ops, locations = bytecode.AppendMakeObject(bytecode.ObjectKindOption, len(e.args), e.location.Bytecode(), ops, locations)
	return ops, locations
}

func (e *Constructor) DataName() ast.FullIdentifier {
	return e.dataName
}

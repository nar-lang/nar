package typed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
	bytecode "nar-compiler/pkg/bytecode"
)

type POption struct {
	*patternBase
	definition *Definition
	args       []Pattern
}

func NewPOption(
	ctx *SolvingContext, loc ast.Location, declaredType Type,
	definition *Definition, args []Pattern,
) (Pattern, error) {
	if len(args) > 255 {
		return nil, common.NewErrorAt(loc, "too many arguments (max 255)")
	}
	return ctx.annotatePattern(&POption{
		patternBase: newPatternBase(loc, declaredType),
		definition:  definition,
		args:        args,
	}), nil
}

func (p *POption) name() ast.DataOptionIdentifier {
	ctor, ok := p.definition.body.(*Constructor)
	if !ok {
		panic("Data option pattern should have a constructor definition.")
	}
	return common.MakeDataOptionIdentifier(ctor.dataName, ctor.optionName)
}

func (p *POption) simplify() simplePattern {
	args := common.Map(func(x Pattern) simplePattern { return x.simplify() }, p.args)
	if dataType, ok := p.type_.(*TData); ok {
		return simpleConstructor{
			Union: dataType,
			Name:  p.name(),
			Args:  args,
		}
	} else {
		panic("Data option pattern should have a data type.")
	}
}

func (p *POption) mapTypes(subst map[uint64]Type) error {
	var err error
	p.type_, err = p.type_.mapTo(subst)
	if err != nil {
		return err
	}
	for _, arg := range p.args {
		err = arg.mapTypes(subst)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *POption) Code(currentModule ast.QualifiedIdentifier) string {
	s := string(p.name())
	if len(p.args) > 0 {
		s += "(" + common.Fold(func(x Pattern, s string) string {
			if s != "" {
				s += ", "
			}
			return s + x.Code(currentModule)
		}, "", p.args) + ")"
	}
	if p.declaredType != nil {
		s += ": " + p.declaredType.Code(currentModule)
	}
	return s
}

func (p *POption) Children() []Statement {
	return append(p.patternBase.Children(), common.Map(func(x Pattern) Statement { return x }, p.args)...)
}

func (p *POption) appendBytecode(ops []bytecode.Op, locations []bytecode.Location, binary *bytecode.Binary, hash *bytecode.BinaryHash) ([]bytecode.Op, []bytecode.Location) {
	var err error
	for _, x := range p.args {
		ops, locations = x.appendBytecode(ops, locations, binary, hash)
		if err != nil {
			return nil, nil
		}
	}
	return bytecode.AppendMakePattern(
		bytecode.PatternKindDataOption,
		string(p.name()),
		uint8(len(p.args)), p.location.Bytecode(), ops, locations, binary, hash)
}

func (p *POption) appendEquations(eqs Equations, loc *ast.Location, localDefs localTypesMap, ctx *SolvingContext, stack []*Definition) (Equations, error) {
	if p.definition == nil {
		return nil, common.NewErrorOf(p, "definition not found")
	}
	defType, err := p.definition.uniqueType(ctx, stack)
	if err != nil {
		return nil, err
	}

	if len(p.args) == 0 {
		eqs = append(eqs, NewEquation(p, p.type_, defType))
	} else {
		eqs = append(eqs, NewEquation(p,
			NewTFunc(p.location, common.Map(func(x Pattern) Type { return x.Type() }, p.args), p.type_),
			defType))
		for _, arg := range p.args {
			var err error
			eqs, err = arg.appendEquations(eqs, loc, localDefs, ctx, stack)
			if err != nil {
				return nil, err
			}
		}
	}

	if p.declaredType != nil {
		eqs = append(eqs, NewEquation(p, p.type_, p.declaredType))
	}
	return eqs, nil
}

func (p *POption) Definition() *Definition {
	return p.definition
}

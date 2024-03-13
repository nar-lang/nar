package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
	bytecode "nar-compiler/pkg/bytecode"
)

type Definition struct {
	id           uint64
	name         ast.Identifier
	nameLocation ast.Location
	location     ast.Location
	params       []Pattern
	body         Expression
	declaredType Type
	type_        Type
	hidden       bool
	ctx          *SolvingContext
	typed        bool
}

func NewDefinition(
	location ast.Location,
	id uint64,
	hidden bool,
	name ast.Identifier,
	nameLocation ast.Location,
) *Definition {
	def := &Definition{
		id:           id,
		name:         name,
		location:     location,
		nameLocation: nameLocation,
		hidden:       hidden,
		ctx:          newSolvingContext(),
	}
	def.type_ = def.ctx.newTypeAnnotation(def)
	return def
}

func (def *Definition) uniqueType(ctx *SolvingContext, stack []*Definition) (Type, error) {
	t := def.declaredType
	if t == nil && def.typed {
		t = def.type_
	}
	if t == nil {
		for _, sd := range stack {
			if sd.id == def.id {
				return sd.type_, nil
			}
		}

		err := def.solveTypes(stack)
		if err != nil {
			return nil, err
		}
		t = def.type_
	}

	return t.makeUnique(ctx, map[uint64]uint64{}), nil
}

func (def *Definition) solveTypes(stack []*Definition) error {
	stack = append(stack, def)
	eqs, err := def.appendEquations(nil, nil, localTypesMap{}, def.ctx, stack)
	if err != nil {
		return err
	}
	eqs = appendUsefulEquations(nil, eqs)

	eqs, err = def.ctx.insertAll(eqs)
	if err != nil {
		return err
	}

	subst := def.ctx.subst()

	err = def.mapTypes(subst)
	if err != nil {
		return err
	}

	stack = stack[:len(stack)-1]
	return nil
}

func (def *Definition) mapTypes(subst map[uint64]Type) error {
	for _, p := range def.params {
		if err := p.mapTypes(subst); err != nil {
			return err
		}
	}
	var err error
	def.type_, err = def.type_.mapTo(subst)
	def.typed = true
	if err != nil {
		return err
	}
	if def.body == nil {
		return nil //common.NewErrorOf(def, "missing body")
	}
	return def.body.mapTypes(subst)
}

func (def *Definition) Children() []Statement {
	return append(common.Map(func(x Pattern) Statement { return x }, def.params),
		def.declaredType, def.body)
}

func (def *Definition) Location() ast.Location {
	return def.location
}

func (def *Definition) SetExpression(body Expression) {
	def.body = body
}

func (def *Definition) SetParams(params []Pattern) {
	def.params = params
}

func (def *Definition) Id() uint64 {
	return def.id
}

func (def *Definition) DeclaredType() Type {
	return def.declaredType
}

func (def *Definition) Params() []Pattern {
	return def.params
}

func (def *Definition) Body() Expression {
	return def.body
}

func (def *Definition) Code(currentModule ast.QualifiedIdentifier) string {
	params := common.Fold(func(x Pattern, s string) string {
		if s != "" {
			s += ", "
		}
		return s + x.Code(currentModule)
	}, "", def.params)
	if params != "" {
		params = "(" + params + ")"
	}
	var typeString string
	switch def.declaredType.(type) {
	case nil:
		break
	case *TFunc:
		typeString = ": " + def.declaredType.(*TFunc).return_.Code(currentModule)
		break
	default:
		typeString = ": " + def.declaredType.Code(currentModule)
		break
	}
	if def.body == nil {
		return fmt.Sprintf("def %s%s%s", def.name, params, typeString)
	}
	return fmt.Sprintf("def %s%s%s = %s", def.name, params, typeString, def.body.Code(currentModule))
}

func (def *Definition) Bytecode(pathId ast.FullIdentifier, modName ast.QualifiedIdentifier, binary *bytecode.Binary, hash *bytecode.BinaryHash) bytecode.Func {
	var ops []bytecode.Op
	var locations []bytecode.Location

	if nc, ok := def.body.(*Call); ok && pathId == nc.name {
		ops, locations = bytecode.AppendCall(string(nc.name), uint8(len(nc.args)), nc.location.Bytecode(), ops, locations, binary, hash)
	} else {
		for i := len(def.params) - 1; i >= 0; i-- {
			p := def.params[i]
			ops, locations = p.appendBytecode(ops, locations, binary, hash)
			ops, locations = bytecode.AppendJump(0, true, p.Location().Bytecode(), ops, locations)
			ops, locations = bytecode.AppendSwapPop(p.Location().Bytecode(), bytecode.SwapPopModePop, ops, locations)
		}
		ops, locations = def.body.appendBytecode(ops, locations, binary, hash)
	}

	return bytecode.Func{
		Name:      hash.HashString(string(common.MakeFullIdentifier(modName, def.name)), binary),
		NumArgs:   uint32(len(def.params)),
		Ops:       ops,
		FilePath:  def.location.FilePath(),
		Locations: locations,
	}
}

func (def *Definition) appendEquations(eqs Equations, loc *ast.Location, localDefs localTypesMap, ctx *SolvingContext, stack []*Definition) (Equations, error) {
	if def.body != nil {
		defType := def.body.Type()

		if len(def.params) > 0 {
			defType = NewTFunc(def.location, common.Map(func(x Pattern) Type { return x.Type() }, def.params), defType)
		}

		l := loc
		if l == nil {
			l = &def.location
		}
		eqs = append(eqs, NewEquation(defType, def.type_, defType))

		if def.declaredType != nil {
			eqs = append(eqs, NewEquation(defType, def.declaredType, defType))
		}
	}

	var err error
	for _, p := range def.params {
		eqs, err = p.appendEquations(eqs, loc, localDefs, ctx, stack)
		if err != nil {
			return nil, err
		}
	}

	if def.body != nil {
		eqs, err = def.body.appendEquations(eqs, loc, localDefs, ctx, stack)
		if err != nil {
			return nil, err
		}
	}
	return eqs, nil
}

func (def *Definition) checkPatterns() error {
	for _, pattern := range def.Params() {
		if err := checkPattern(pattern); err != nil {
			return err
		}
	}
	if def.body != nil {
		return def.body.checkPatterns()
	}
	return nil
}

func (def *Definition) SetDeclaredType(declaredType Type) {
	def.declaredType = declaredType
}

func (def *Definition) SolvingContext() *SolvingContext {
	return def.ctx
}

func (def *Definition) Name() ast.Identifier {
	return def.name
}

func (def *Definition) NameLocation() ast.Location {
	return def.nameLocation
}

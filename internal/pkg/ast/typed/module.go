package typed

import (
	"fmt"
	"oak-compiler/internal/pkg/ast"
	"oak-compiler/internal/pkg/common"
)

type Definition struct {
	Id          uint64
	Name        ast.Identifier
	Location    ast.Location
	Params      []Pattern
	Expression  Expression
	DefinedType Type
	Hidden      bool
}

func (d *Definition) GetType() Type {
	if d.Expression == nil {
		return d.DefinedType
	}

	defType := d.Expression.GetType()

	if len(d.Params) > 0 {
		defType = &TFunc{
			Location: d.Location,
			Params:   common.Map(func(x Pattern) Type { return x.GetType() }, d.Params),
			Return:   defType,
		}
	}

	return defType
}

func (d *Definition) String() string {
	return fmt.Sprintf("Def([%v], %v)", common.Join(d.Params, ", "), d.Expression)
}

type Module struct {
	Name         ast.QualifiedIdentifier
	Dependencies []ast.QualifiedIdentifier
	Definitions  []*Definition
}

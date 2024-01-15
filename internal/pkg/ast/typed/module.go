package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
)

type Definition struct {
	Id           uint64
	Name         ast.Identifier
	Location     ast.Location
	Params       []Pattern
	Expression   Expression
	DeclaredType Type
	Hidden       bool
}

func (d *Definition) GetLocation() ast.Location {
	return d.Location
}

func (d *Definition) String() string {
	return fmt.Sprintf("Definition(%v,%v,%v,%v,%v)", d.Name, d.Params, d.Expression, d.DeclaredType, d.Location)
}

func (d *Definition) GetType() Type {
	if d.Expression == nil {
		return d.DeclaredType
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

type Module struct {
	Name         ast.QualifiedIdentifier
	Location     ast.Location
	Dependencies map[ast.QualifiedIdentifier][]ast.Identifier
	Definitions  []*Definition
}

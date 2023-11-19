package typed

import (
	"fmt"
	"oak-compiler/internal/pkg/ast"
)

type Definition struct {
	Id          uint64
	Pattern     Pattern
	Expression  Expression
	DefinedType Type
	Hidden      bool
}

func (d *Definition) GetType() Type {
	if d.Expression == nil {
		return d.DefinedType
	}
	return d.Expression.GetType()
}

func (d *Definition) String() string {
	return fmt.Sprintf("Def(%v,%v)", d.Pattern, d.Expression)
}

type Module struct {
	Name         ast.QualifiedIdentifier
	Dependencies []ast.QualifiedIdentifier
	Definitions  map[ast.Identifier]*Definition
}

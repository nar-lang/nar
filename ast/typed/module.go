package typed

import (
	"fmt"
	"oak-compiler/ast"
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
	Path        string
	Name        ast.QualifiedIdentifier
	DepPaths    []string
	Definitions map[ast.Identifier]*Definition
}

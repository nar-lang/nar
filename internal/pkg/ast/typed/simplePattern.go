package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
)

type simplePattern interface {
	fmt.Stringer
	_simplePattern()
}

type simpleAnything struct{}

func (simpleAnything) _simplePattern() {}

func (simpleAnything) String() string {
	return "_"
}

type simpleLiteral struct {
	Literal ast.ConstValue
}

func (simpleLiteral) _simplePattern() {}

func (p simpleLiteral) String() string {
	return p.Literal.Code("")
}

type simpleConstructor struct {
	Union *TData
	Name  ast.DataOptionIdentifier
	Args  []simplePattern
}

func (simpleConstructor) _simplePattern() {}

func (c simpleConstructor) String() string {
	params := common.Join(c.Args, ", ")
	if params != "" {
		params = fmt.Sprintf("(%s)", params)
	}
	return fmt.Sprintf("%s%s", c.Name, params)
}

func (c simpleConstructor) Option() (*DataOption, error) {
	for _, o := range c.Union.options {
		if o.name == c.Name {
			return o, nil
		}
	}
	return nil, common.NewCompilerError("option not found")
}

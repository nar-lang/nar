package typed

import (
	"fmt"
	"oak-compiler/internal/pkg/ast"
	"oak-compiler/internal/pkg/common"
	"strings"
)

type Type interface {
	fmt.Stringer
	_type()
	GetLocation() ast.Location
}

type TFunc struct {
	ast.Location
	Params []Type
	Return Type
}

func (*TFunc) _type() {}

func (t *TFunc) GetLocation() ast.Location {
	return t.Location
}

func (t *TFunc) String() string {
	return fmt.Sprintf("(%v): %v", common.Join(t.Params, ", "), t.Return)
}

type TRecord struct {
	ast.Location
	Fields map[ast.Identifier]Type
}

func (*TRecord) _type() {}

func (t *TRecord) GetLocation() ast.Location {
	return t.Location
}

func (t *TRecord) String() string {
	sb := strings.Builder{}
	sb.WriteString("{")
	c := len(t.Fields)
	for n, v := range t.Fields {
		sb.WriteString(fmt.Sprintf("%s:%v", n, v))
		c--
		if c > 0 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("}")
	return sb.String()
}

type TTuple struct {
	ast.Location
	Items []Type
}

func (*TTuple) _type() {}

func (t *TTuple) GetLocation() ast.Location {
	return t.Location
}

func (t *TTuple) String() string {
	return fmt.Sprintf("( %v )", common.Join(t.Items, ", "))
}

type TExternal struct {
	ast.Location
	Name ast.ExternalIdentifier
	Args []Type
}

func (*TExternal) _type() {}

func (t *TExternal) GetLocation() ast.Location {
	return t.Location
}

func (t *TExternal) String() string {
	tp := common.Join(t.Args, ", ")
	if tp != "" {
		tp = "[" + tp + "]"
	}
	return fmt.Sprintf("%v%v", t.Name, tp)
}

type DataOption struct {
	Name   ast.DataOptionIdentifier
	Values []Type
}

func (d DataOption) String() string {
	return fmt.Sprintf("%s(%v)", d.Name, common.Join(d.Values, ", "))
}

type TData struct {
	ast.Location
	Name    ast.ExternalIdentifier
	Options []DataOption
	Args    []Type
}

func (*TData) _type() {}

func (t *TData) GetLocation() ast.Location {
	return t.Location
}

func (t *TData) String() string {
	return fmt.Sprintf("%s(%v)", t.Name, common.Join(t.Options, ", "))
}

type TUnbound struct {
	ast.Location
	Index      uint64
	Constraint common.Constraint
}

func (*TUnbound) _type() {}

func (t *TUnbound) GetLocation() ast.Location {
	return t.Location
}

func (t *TUnbound) String() string {
	return fmt.Sprintf("_%d", t.Index)
}

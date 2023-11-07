package typed

import (
	"fmt"
	"oak-compiler/ast"
	"oak-compiler/common"
	"strings"
)

type Type interface {
	fmt.Stringer
	_type()
	EqualsTo(o Type) bool
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

func (t *TFunc) EqualsTo(o Type) bool {
	if y, ok := o.(*TFunc); ok {
		if !t.Return.EqualsTo(y.Return) {
			return false
		}
		if len(t.Params) != len(y.Params) {
			return false
		}
		for i, a := range t.Params {
			if !a.EqualsTo(y.Params[i]) {
				return false
			}
		}
		return true
	}
	return false
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

func (t *TRecord) EqualsTo(o Type) bool {
	if y, ok := o.(*TRecord); ok {
		if len(t.Fields) != len(y.Fields) {
			return false
		}
		for i, a := range t.Fields {
			b, ok := y.Fields[i]
			if !ok {
				return false
			}
			if !a.EqualsTo(b) {
				return false
			}
		}
		return true
	}
	return false
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

func (t *TTuple) EqualsTo(o Type) bool {
	if y, ok := o.(*TTuple); ok {
		if len(t.Items) != len(y.Items) {
			return false
		}
		for i, a := range t.Items {
			if !a.EqualsTo(y.Items[i]) {
				return false
			}
		}
		return true
	}
	return false
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
	return fmt.Sprintf("%s%s", t.Name, tp)
}

func (t *TExternal) EqualsTo(o Type) bool {
	if y, ok := o.(*TExternal); ok {
		if t.Name != y.Name {
			return false
		}
		if len(t.Args) != len(y.Args) {
			return false
		}
		for i, a := range t.Args {
			if !a.EqualsTo(y.Args[i]) {
				return false
			}
		}
		return true
	}
	return false
}

type TUnbound struct {
	ast.Location
	Index uint64
}

func (*TUnbound) _type() {}

func (t *TUnbound) GetLocation() ast.Location {
	return t.Location
}

func (t *TUnbound) String() string {
	return fmt.Sprintf("u%d", t.Index)
}

func (t *TUnbound) EqualsTo(o Type) bool {
	if y, ok := o.(*TUnbound); ok {
		return t.Index == y.Index
	}
	return false
}

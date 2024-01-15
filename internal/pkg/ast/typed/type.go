package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
	"strings"
)

type UnboundMap map[uint64]string

type Type interface {
	_type()
	GetLocation() ast.Location
	ToString(m UnboundMap, currentModule ast.QualifiedIdentifier) string
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

func (t *TFunc) ToString(m UnboundMap, currentModule ast.QualifiedIdentifier) string {
	return fmt.Sprintf("(%s): %s", common.Fold(
		func(x Type, s string) string {
			if s != "" {
				s += ", "
			}
			return s + x.ToString(m, "")
		},
		"", t.Params),
		t.Return.ToString(m, ""))
}

type TRecord struct {
	ast.Location
	Fields            map[ast.Identifier]Type
	MayHaveMoreFields bool
}

func (*TRecord) _type() {}

func (t *TRecord) GetLocation() ast.Location {
	return t.Location
}

func (t *TRecord) ToString(m UnboundMap, currentModule ast.QualifiedIdentifier) string {
	sb := strings.Builder{}
	sb.WriteString("{")
	c := len(t.Fields)
	for n, v := range t.Fields {
		sb.WriteString(fmt.Sprintf("%s:%s", n, v.ToString(m, "")))
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

func (t *TTuple) ToString(m UnboundMap, currentModule ast.QualifiedIdentifier) string {
	return fmt.Sprintf("( %v )", common.Fold(func(x Type, s string) string {
		if s != "" {
			s += ", "
		}
		return s + x.ToString(m, "")
	}, "", t.Items))
}

type TNative struct {
	ast.Location
	Name ast.FullIdentifier
	Args []Type
}

func (*TNative) _type() {}

func (t *TNative) GetLocation() ast.Location {
	return t.Location
}

func (t *TNative) ToString(m UnboundMap, currentModule ast.QualifiedIdentifier) string {
	tp := common.Fold(func(x Type, s string) string {
		if s != "" {
			s += ", "
		}
		return s + x.ToString(m, "")
	}, "", t.Args)
	if tp != "" {
		tp = "[" + tp + "]"
	}
	s := string(t.Name)
	if currentModule != "" && strings.HasPrefix(s, string(currentModule)) {
		s = s[len(currentModule)+1:]
	}
	return fmt.Sprintf("%s%s", s, tp)
}

type DataOption struct {
	Name   ast.DataOptionIdentifier
	Values []Type
}

func (d DataOption) String() string {
	return fmt.Sprintf("%s(%v)", d.Name, len(d.Values))
}

type TData struct {
	ast.Location
	Name    ast.FullIdentifier
	Options []DataOption
	Args    []Type
}

func (*TData) _type() {}

func (t *TData) GetLocation() ast.Location {
	return t.Location
}

func (t *TData) ToString(m UnboundMap, currentModule ast.QualifiedIdentifier) string {
	s := string(t.Name)
	if currentModule != "" && strings.HasPrefix(s, string(currentModule)) {
		s = s[len(currentModule)+1:]
	}
	tp := common.Fold(func(x Type, s string) string {
		if s != "" {
			s += ", "
		}
		return s + x.ToString(m, "")
	}, "", t.Args)
	if tp != "" {
		tp = "[" + tp + "]"
	}
	return s + tp
}

type TUnbound struct {
	ast.Location
	Index      uint64
	Constraint common.Constraint
	GivenName  ast.Identifier
}

func (*TUnbound) _type() {}

func (t *TUnbound) GetLocation() ast.Location {
	return t.Location
}

func (t *TUnbound) ToString(m UnboundMap, currentModule ast.QualifiedIdentifier) string {
	if t.GivenName != "" {
		m[t.Index] = string(t.GivenName)
		return string(t.GivenName)
	}
	if n, ok := m[t.Index]; ok {
		return n
	}
	i := uint64(0)
	for {
		if _, ok := m[i]; ok {
			continue
		}
		break
	}
	v := string(rune(int('a') + int(i)))
	m[t.Index] = v
	return v
}

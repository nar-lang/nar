package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
	"strings"
)

type TData struct {
	*typeBase
	name    ast.FullIdentifier
	options []*DataOption
	args    []Type
}

func NewTData(loc ast.Location, name ast.FullIdentifier, args []Type, options []*DataOption) Type {
	return &TData{
		typeBase: newTypeBase(loc),
		name:     name,
		options:  options,
		args:     args,
	}
}

func (t *TData) makeUnique(ctx *SolvingContext, ubMap map[uint64]uint64) Type {
	return NewTData(t.location, t.name, common.Map(func(x Type) Type { return x.makeUnique(ctx, ubMap) }, t.args), t.options)
}

func (t *TData) merge(other Type, loc ast.Location) (Equations, error) {
	if o, ok := other.(*TData); ok {
		if o.name == t.name {
			if len(t.args) == len(o.args) {
				var eqs Equations
				for i, a := range t.args {
					eqs = append(eqs, NewEquationBestLoc(a, o.args[i], loc))
				}
				return eqs, nil
			}
		}
	}
	return nil, newTypeMatchError(loc, t, other)
}

func (t *TData) mapTo(subst map[uint64]Type) (Type, error) {
	for i, a := range t.args {
		if x, err := a.mapTo(subst); err != nil {
			return nil, err
		} else {
			t.args[i] = x
		}
	}
	return t, nil
}

func (t *TData) equalsTo(other Type, req map[ast.FullIdentifier]struct{}) bool {
	ty, oky := other.(*TData)
	if oky {
		if t.name != ty.name {
			return false
		}
		if req != nil {
			if _, ok := req[t.name]; ok {
				return true
			}
		}
		req = map[ast.FullIdentifier]struct{}{}
		req[t.name] = struct{}{}
		if len(t.args) != len(ty.args) {
			return false
		}
		for i, a := range t.args {
			if !a.equalsTo(ty.args[i], req) {
				return false
			}
		}
		return true
	}
	return false
}

func (t *TData) Children() []Statement {
	return append(
		common.Map(func(x Type) Statement { return x }, t.args),
		common.FlatMap(func(x *DataOption) []Statement { return x.Children() }, t.options)...)
}

func (t *TData) Code(currentModule ast.QualifiedIdentifier) string {
	s := string(t.name)
	if currentModule != "" && strings.HasPrefix(s, string(currentModule)) {
		s = s[len(currentModule)+1:]
	}
	tp := common.Fold(func(x Type, s string) string {
		if s != "" {
			s += ", "
		}
		return s + x.Code("")
	}, "", t.args)
	if tp != "" {
		tp = "[" + tp + "]"
	}
	return s + tp
}

func (t *TData) SetOptions(options []*DataOption) {
	t.options = options
}

type DataOption struct {
	name   ast.DataOptionIdentifier
	values []Type
}

func NewDataOption(name ast.DataOptionIdentifier, values []Type) *DataOption {
	return &DataOption{
		name:   name,
		values: values,
	}
}

func (d *DataOption) String() string {
	return fmt.Sprintf("%s(%d)", d.name, len(d.values))
}

func (d *DataOption) Children() []Statement {
	return common.Map(func(x Type) Statement { return x }, d.values)
}

package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
	"strings"
)

type TRecord struct {
	*typeBase
	fields            map[ast.Identifier]Type
	mayHaveMoreFields bool
}

func NewTRecord(loc ast.Location, fields map[ast.Identifier]Type, mayHaveMoreFields bool) Type {
	return &TRecord{
		typeBase:          newTypeBase(loc),
		fields:            fields,
		mayHaveMoreFields: mayHaveMoreFields,
	}
}

func (t *TRecord) makeUnique(ctx *SolvingContext, ubMap map[uint64]uint64) Type {
	nf := make(map[ast.Identifier]Type, len(t.fields))
	for n, f := range t.fields {
		nf[n] = f.makeUnique(ctx, ubMap)
	}
	return NewTRecord(t.location, nf, t.mayHaveMoreFields)
}

func (t *TRecord) merge(other Type, loc ast.Location) (Equations, error) {
	if o, ok := other.(*TRecord); ok {
		var eqs Equations
		for n, f := range t.fields {
			if of, ok := o.fields[n]; ok {
				eqs = append(eqs, NewEquationBestLoc(f, of, loc))
			} else if !o.mayHaveMoreFields {
				return nil, common.Error{Location: loc, Message: fmt.Sprintf("record missing field `%s`", n)}
			}
		}
		for n := range o.fields {
			if _, ok := t.fields[n]; !ok && !t.mayHaveMoreFields {
				return nil, common.Error{Location: loc, Message: fmt.Sprintf("record missing field `%s`", n)}
			}
		}
		return eqs, nil
	}
	return nil, newTypeMatchError(loc, t, other)
}

func (t *TRecord) mapTo(subst map[uint64]Type) (Type, error) {
	for n, f := range t.fields {
		if x, err := f.mapTo(subst); err != nil {
			return nil, err
		} else {
			t.fields[n] = x
		}
	}
	return t, nil
}

func (t *TRecord) equalsTo(other Type, req map[ast.FullIdentifier]struct{}) bool {
	ty, oky := other.(*TRecord)
	if oky {
		if len(t.fields) != len(ty.fields) {
			return false
		}
		for n, fx := range t.fields {
			if fy, ok := ty.fields[n]; !ok {
				return false
			} else if !fx.equalsTo(fy, req) {
				return false
			}
		}
		return true
	}
	return false
}

func (t *TRecord) Children() []Statement {
	return common.Map(func(x Type) Statement { return x }, common.Values(t.fields))
}

func (t *TRecord) Code(currentModule ast.QualifiedIdentifier) string {
	sb := strings.Builder{}
	sb.WriteString("{")
	c := len(t.fields)
	for n, v := range t.fields {
		sb.WriteString(fmt.Sprintf("%s:%s", n, v.Code("")))
		c--
		if c > 0 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("}")
	return sb.String()
}

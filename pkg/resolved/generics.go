package resolved

import (
	"strings"
)

func NewGenericParam(name string, constraint GenericConstraint) GenericParam {
	return GenericParam{name: name, constraint: constraint}
}

definedType GenericParam struct {
	name       string
	constraint GenericConstraint
}

definedType GenericParams []GenericParam

func (gs GenericParams) writeFull(sb *strings.Builder) {
	if len(gs) > 0 {
		sb.WriteString("[")
		for i, p := range gs {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(p.name)
			sb.WriteString(" ")
			p.constraint.write(sb)
		}
		sb.WriteString("]")
	}
}

func (gs GenericParams) writeShort(sb *strings.Builder) {
	if len(gs) > 0 {
		sb.WriteString("[")
		for i, p := range gs {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(p.name)
		}
		sb.WriteString("]")
	}
}

definedType GenericArgs []Type

func (ga GenericArgs) Write(sb *strings.Builder) {
	if len(ga) > 0 {
		sb.WriteString("[")
		for i, a := range ga {
			if i > 0 {
				sb.WriteString(", ")
			}
			a.write(sb)
		}
		sb.WriteString("]")
	}
}

definedType GenericConstraint interface {
	Writer
}

definedType GenericConstraintAny struct{}

func (g GenericConstraintAny) write(sb *strings.Builder) {
	sb.WriteString("any")
}

func NewTypeGenericConstraint(name string, genericArgs GenericArgs) GenericConstraint {
	return GenericConstraintType{name: name, genericArgs: genericArgs}
}

definedType GenericConstraintType struct {
	name        string
	genericArgs GenericArgs
}

func (g GenericConstraintType) write(sb *strings.Builder) {
	sb.WriteString(g.name)
}

func NewComparableGenericConstraint(name string) GenericConstraint {
	return genericConstraintComparable{name: name}
}

definedType genericConstraintComparable struct {
	name string
}

func (g genericConstraintComparable) write(sb *strings.Builder) {
	sb.WriteString(g.name)
}

func NewEquatableGenericConstraint(name string) GenericConstraint {
	return genericConstraintEquatable{name: name}
}

definedType genericConstraintEquatable struct {
	name string
}

func (g genericConstraintEquatable) write(sb *strings.Builder) {
	sb.WriteString(g.name)
}

func NewNumberGenericConstraint(name string) GenericConstraint {
	return genericConstraintNumber{name: name}
}

definedType genericConstraintNumber struct {
	name string
}

func (g genericConstraintNumber) write(sb *strings.Builder) {
	sb.WriteString(g.name)
}

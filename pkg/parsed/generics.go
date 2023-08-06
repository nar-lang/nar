package parsed

import (
	"fmt"
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
	"strings"
)

func NewGenericParam(c misc.Cursor, modName ModuleFullName, name string, constraint GenericConstraint) GenericParam {
	return GenericParam{cursor: c, modName: modName, name: name, constraint: constraint}
}

type GenericParam struct {
	name       string
	constraint GenericConstraint
	cursor     misc.Cursor
	modName    ModuleFullName
}

func (p GenericParam) Name() string {
	return p.name
}

func (p GenericParam) Resolve(md *Metadata) (resolved.GenericParam, error) {
	resolvedConstraint, err := p.constraint.Resolve(p.cursor, md)
	if err != nil {
		return resolved.GenericParam{}, err
	}
	return resolved.NewGenericParam(p.name, resolvedConstraint), nil
}

type GenericParams []GenericParam

func (gs GenericParams) toArgs() GenericArgs {
	var args GenericArgs
	for _, p := range gs {
		args = append(args, NewGenericNameType(p.cursor, p.modName, p.name))
	}

	return args
}

func (gs GenericParams) Resolve(md *Metadata) (resolved.GenericParams, error) {
	var params resolved.GenericParams
	for _, p := range gs {
		resolvedParam, err := p.Resolve(md)
		if err != nil {
			return nil, err
		}
		params = append(params, resolvedParam)
	}
	return params, nil
}

type GenericArgs []Type

func (a GenericArgs) resolve(cursor misc.Cursor, md *Metadata) (resolved.GenericArgs, error) {
	var args resolved.GenericArgs
	for _, arg := range a {
		resolvedArg, err := arg.resolve(cursor, md)
		if err != nil {
			return nil, err
		}
		args = append(args, resolvedArg)
	}
	return args, nil
}

func (a GenericArgs) mapGenerics(gm genericsMap) GenericArgs {
	var args GenericArgs
	for _, arg := range a {
		args = append(args, arg.mapGenerics(gm))
	}
	return args
}

func (a GenericArgs) equalsTo(o GenericArgs, ignoreGenerics bool, md *Metadata) bool {
	if len(a) != len(o) {
		return false
	}

	for i, x := range a {
		if !typesEqual(x, o[i], ignoreGenerics, md) {
			return false
		}
	}

	return true
}

func (a GenericArgs) String() string {
	sb := strings.Builder{}
	if len(a) != 0 {
		sb.WriteString("[")
		for i, g := range a {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(g.String())
		}
		sb.WriteString("]")
	}
	return sb.String()
}

func (a GenericArgs) extractGenerics(other GenericArgs, gm genericsMap) {
	if len(a) == len(other) {
		for i, g := range a {
			if n, ok := g.(typeGenericNotResolved); ok {
				gm[n.Name] = other[i]
			}
		}
	}
}

type GenericConstraint interface {
	Resolve(cursor misc.Cursor, md *Metadata) (resolved.GenericConstraint, error)
}

type GenericConstraintAny struct {
	GenericConstraintAny__ int
}

func (g GenericConstraintAny) Resolve(cursor misc.Cursor, md *Metadata) (resolved.GenericConstraint, error) {
	return resolved.GenericConstraintAny{}, nil
}

type GenericConstraintType struct {
	GenericConstraintType__ int
	Name                    string
	GenericArgs             GenericArgs
}

func (g GenericConstraintType) Resolve(cursor misc.Cursor, md *Metadata) (resolved.GenericConstraint, error) {
	resolvedArgs, err := g.GenericArgs.resolve(cursor, md)
	if err != nil {
		return nil, err
	}
	return resolved.NewTypeGenericConstraint(g.Name, resolvedArgs), nil
}

type GenericConstraintComparable struct {
	GenericConstraintComparable__ int
}

func (g GenericConstraintComparable) Resolve(cursor misc.Cursor, md *Metadata) (resolved.GenericConstraint, error) {
	return resolved.NewComparableGenericConstraint(makeSpecialGenericName("runtime.Comparable", md)), nil
}

type GenericConstraintEquatable struct {
	GenericConstraintEquatable__ int
}

func (g GenericConstraintEquatable) Resolve(cursor misc.Cursor, md *Metadata) (resolved.GenericConstraint, error) {
	return resolved.NewEquatableGenericConstraint(makeSpecialGenericName("runtime.Equatable", md)), nil
}

type GenericConstraintCombined struct {
	GenericConstraintCombined__ int
	Constraints                 []GenericConstraint
}

func (g GenericConstraintCombined) Resolve(cursor misc.Cursor, md *Metadata) (resolved.GenericConstraint, error) {
	var gs resolved.GenericConstraintCombined

	for _, x := range g.Constraints {
		resolvedConstraint, err := x.Resolve(cursor, md)
		if err != nil {
			return nil, err
		}
		gs = append(gs, resolvedConstraint)
	}

	return gs, nil
}

func makeSpecialGenericName(name string, md *Metadata) string {
	basicsModule := ModuleFullName{packageName: kCoreFullPackageName, moduleName: kBasicsModuleName}
	if md.currentModuleName() == basicsModule {
		return name
	}
	return fmt.Sprintf("%s.%s", md.ImportModuleAliases[basicsModule], kCoreFullPackageName)
}

type genericsMap map[string]Type

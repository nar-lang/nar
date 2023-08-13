package parsed

import (
	"golang.org/x/exp/maps"
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
	resolvedConstraint, err := p.constraint.resolve(p.cursor, md)
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

func (gs GenericParams) resolve(md *Metadata) (resolved.GenericParams, error) {
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

func (gs GenericParams) byName(name string) (GenericParam, bool) {
	for _, p := range gs {
		if p.name == name {
			return p, true
		}
	}
	return GenericParam{}, false
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

func (a GenericArgs) extractGenerics(other GenericArgs) genericsMap {
	gm := genericsMap{}
	if len(a) == len(other) {
		for i, g := range a {
			if n, ok := g.(typeGenericNotResolved); ok {
				if _, ok := other[i].(typeGenericNotResolved); !ok {
					gm[n.Name] = other[i]
				}
			}
		}
	}
	return gm
}

type genericsMap map[string]Type

func mergeGenericMaps(dst genericsMap, src genericsMap) genericsMap {
	if len(src) == 0 {
		return dst
	}
	if len(dst) == 0 {
		return src
	}
	res := genericsMap{}
	maps.Copy(res, dst)
	for name, type_ := range src {
		if existing, ok := res[name]; ok {
			gm := type_.extractGenerics(existing)
			type_ = type_.mapGenerics(gm)
			gm = existing.extractGenerics(type_)
			type_ = existing.mapGenerics(gm)
		}
		res[name] = type_
	}

	return res
}

func (gm genericsMap) mapSelf() genericsMap {
	for n, g := range gm {
		if _, ok := g.(typeGenericNotResolved); ok {
			gm[n] = g.mapGenerics(gm)
		}
	}
	return gm
}

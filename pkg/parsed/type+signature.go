package parsed

import (
	"fmt"
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewSignatureType(c misc.Cursor, modName ModuleFullName, paramType, ret Type, param Parameter) Type {
	return typeSignature{
		Param:      param,
		ParamType:  paramType,
		ReturnType: ret,
		typeBase:   typeBase{cursor: c, moduleName: modName},
	}
}

type typeSignature struct {
	typeBase
	Param      Parameter
	ParamType  Type
	ReturnType Type
}

func (t typeSignature) extractGenerics(other Type) genericsMap {
	if ts, ok := other.(typeSignature); ok {
		return mergeGenericMaps(t.ParamType.extractGenerics(ts.ParamType), t.ReturnType.extractGenerics(ts.ReturnType))
	}
	return nil
}

func (t typeSignature) equalsTo(other Type, ignoreGenerics bool, md *Metadata) bool {
	o, ok := other.(typeSignature)

	return ok &&
		typesEqual(o.ParamType, t.ParamType, ignoreGenerics, md) &&
		typesEqual(o.ReturnType, t.ReturnType, ignoreGenerics, md)
}

func (t typeSignature) String() string {
	return fmt.Sprintf("%s -> %s", t.ParamType.String(), t.ReturnType.String())
}

func (t typeSignature) getGenerics() GenericArgs {
	return nil
}

func (t typeSignature) mapGenerics(gm genericsMap) Type {
	t.ParamType = t.ParamType.mapGenerics(gm)
	t.ReturnType = t.ReturnType.mapGenerics(gm)
	return t
}

func (t typeSignature) dereference(md *Metadata) (Type, error) {
	return t, nil
}

func (t typeSignature) nestedDefinitionNames() []string {
	return nil
}

func (t typeSignature) unpackNestedDefinitions(def Definition) []Definition {
	return nil
}

func (t typeSignature) resolveWithRefName(
	cursor misc.Cursor, refName string, generics GenericArgs, md *Metadata,
) (resolved.Type, error) {
	resolvedParamType, err := t.ParamType.resolve(cursor, md)
	if err != nil {
		return nil, err
	}
	resolvedReturnType, err := t.ReturnType.resolve(cursor, md)
	if err != nil {
		return nil, err
	}
	resolvedGenerics, err := generics.resolve(cursor, md)
	if err != nil {
		return nil, err
	}
	resolvedParam, err := t.Param.resolve(t.ParamType, md)
	if err != nil {
		return nil, err
	}

	return resolved.NewRefSignatureType(
		refName, resolvedGenerics, resolvedParam, resolvedParamType, resolvedReturnType,
	), nil
}

func (t typeSignature) resolve(cursor misc.Cursor, md *Metadata) (resolved.Type, error) {
	resolvedParamType, err := t.ParamType.resolve(cursor, md)
	if err != nil {
		return nil, err
	}
	resolvedReturnType, err := t.ReturnType.resolve(cursor, md)
	if err != nil {
		return nil, err
	}

	var resolvedParam resolved.Parameter
	if t.Param == nil {
		resolvedParam = resolved.NewEmptyParameter()
	} else {
		resolvedParam, err = t.Param.resolve(t.ParamType, md)
		if err != nil {
			return nil, err
		}
	}
	return resolved.NewSignatureType(resolvedParam, resolvedParamType, resolvedReturnType), nil
}

func (t typeSignature) typeWithArgs(numArgs int) Type {
	x := Type(t)
	for i := 0; i < numArgs; i++ {
		s, ok := x.(typeSignature)
		if !ok {
			panic("expected signature")
		}
		x = s.ReturnType
	}
	return x
}

func (t typeSignature) flatten(nParams int) ([]Type, Type) {
	var params []Type
	tx := Type(t)
	for i := 0; i < nParams; i++ {
		params = append(params, tx.(typeSignature).ParamType)
		tx = tx.(typeSignature).ReturnType
	}
	return params, tx
}

func (t typeSignature) flattenDefinition() ([]Type, Type) {
	var params []Type
	tx := Type(t)
	for {
		if s, ok := tx.(typeSignature); !ok || s.Param == nil {
			break
		}
		params = append(params, tx.(typeSignature).ParamType)
		tx = tx.(typeSignature).ReturnType
	}
	return params, tx
}

func (t typeSignature) extractLocals(type_ Type, md *Metadata) error {
	dt, err := type_.dereference(md)
	if err != nil {
		return err
	}

	signature, ok := dt.(typeSignature)
	if !ok {
		return misc.NewError(t.cursor, "expected function signature, got %s", type_)
	}
	err = t.Param.extractLocals(signature.ParamType, md)
	if err != nil {
		return err
	}
	return t.ReturnType.extractLocals(signature.ReturnType, md)
}

func collapseSignature(params []Type, ret Type) typeSignature {
	ts := typeSignature{ReturnType: ret}
	for i := len(params) - 1; i >= 0; i++ {
		ts.ParamType = params[i]
		if i > 0 {
			ts.ReturnType = ts
		}
	}
	return ts
}

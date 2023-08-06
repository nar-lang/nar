package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewFuncDefinition(
	c misc.Cursor,
	address DefinitionAddress, genericParams GenericParams,
	hidden, extern bool, type_ Type, expression Expression,
) Definition {
	return definitionFunc{
		definitionBase: definitionBase{
			Address:       address,
			GenericParams: genericParams,
			Hidden:        hidden,
			Extern:        extern,
			cursor:        c,
		},
		Expression: expression,
		Type:       type_,
	}
}

type definitionFunc struct {
	DefinitionFunc__ int
	definitionBase
	Expression Expression
	Type       Type
}

func (def definitionFunc) precondition(md *Metadata) (Definition, error) {
	if def.Extern {
		return def, nil
	}
	gm, err := def.getGenericsMap(def.cursor, def.GenericParams.toArgs())
	if err != nil {
		return nil, err
	}
	_, err = def.injectParameters(gm, md)
	if err != nil {
		return nil, err
	}
	def.Expression, err = def.Expression.precondition(md)
	if err != nil {
		return nil, err
	}
	return def, nil
}

func (def definitionFunc) getType(cursor misc.Cursor, generics GenericArgs, md *Metadata) (Type, GenericArgs, error) {
	gm, err := def.getGenericsMap(cursor, generics)
	if err != nil {
		return nil, nil, err
	}
	return def.Type.mapGenerics(gm), def.GenericParams.toArgs().mapGenerics(gm), nil
}

func (def definitionFunc) nestedDefinitionNames() []string {
	return nil
}

func (def definitionFunc) unpackNestedDefinitions() []Definition {
	return nil
}

func (def definitionFunc) resolveName(misc.Cursor, *Metadata) (string, error) {
	return def.Address.moduleFullName.moduleName + "_" + def.Name(), nil
}

func (def definitionFunc) resolve(md *Metadata) (resolved.Definition, bool, error) {
	if def.Extern {
		return nil, false, nil
	}

	gm, err := def.getGenericsMap(def.cursor, def.GenericParams.toArgs())
	if err != nil {
		return nil, false, err
	}

	returnType, err := def.injectParameters(gm, md)
	if err != nil {
		return nil, false, err
	}

	def.Expression, _, err = def.Expression.setType(returnType, genericsMap{}, md)
	if err != nil {
		return nil, false, err
	}

	resolvedExpression, err := def.Expression.resolve(md)
	if err != nil {
		return nil, false, err
	}

	resolvedName, err := def.resolveName(def.cursor, md)
	if err != nil {
		return nil, false, err
	}

	fnType := def.Type.mapGenerics(gm)
	if err != nil {
		return nil, false, err
	}
	dt, err := fnType.dereference(md)
	if err != nil {
		return nil, false, err
	}

	if _, ok := dt.(typeSignature); !ok {
		fnType = typeSignature{
			ParamName:  "x",
			ParamType:  typeVoid{},
			ReturnType: fnType,
			typeBase: typeBase{
				cursor:     misc.Cursor{},
				moduleName: md.currentModuleName(),
			},
		}
	}

	resolvedType, err := fnType.resolve(def.cursor, md)

	resolvedParams, err := def.GenericParams.Resolve(md)
	if err != nil {
		return nil, false, err
	}

	return resolved.NewFuncDefinition(resolvedName, resolvedParams, resolvedType, resolvedExpression), true, nil
}

func (def definitionFunc) injectParameters(gm genericsMap, md *Metadata) (Type, error) {
	md.CurrentDefinition = def
	md.LocalVars = map[string]Type{}
	defType := def.Type.mapGenerics(gm)
	for {
		dt, err := defType.dereference(md)
		if err != nil {
			return nil, err
		}
		if signature, ok := dt.(typeSignature); ok && signature.ParamName != "" {
			md.LocalVars[signature.ParamName] = signature.ParamType
			defType = signature.ReturnType
		} else {
			break
		}
	}
	return defType, nil
}

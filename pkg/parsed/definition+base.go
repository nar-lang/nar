package parsed

import "oak-compiler/pkg/misc"

type definitionBase struct {
	Address       DefinitionAddress
	GenericParams GenericParams
	Hidden        bool
	Extern        bool
	cursor        misc.Cursor
}

func (def definitionBase) isHidden() bool {
	return def.Hidden
}

func (def definitionBase) isExtern() bool {
	return def.Extern
}

func (def definitionBase) Name() string {
	return def.Address.definitionName
}

func (def definitionBase) getGenerics() GenericParams {
	return def.GenericParams
}

func (def definitionBase) getAddress() DefinitionAddress {
	return def.Address
}

func (def definitionBase) getGenericsMap(cursor misc.Cursor, args GenericArgs, resolved bool) (genericsMap, error) {
	var gm genericsMap
	if len(def.GenericParams) != 0 {
		gm = genericsMap{}
		if len(args) != 0 {
			if len(def.GenericParams) != len(args) {
				return nil, misc.NewError(
					cursor, "expected %d generic arguments, got %d", len(def.GenericParams), len(args),
				)
			}
		} else {
			for _, p := range def.GenericParams {
				if resolved {
					args = append(args, typeGenericName{Name: p.name})
				} else {
					args = append(args, typeGenericNotResolved{Name: p.name})
				}
			}
		}

		for i, gp := range def.GenericParams {
			gm[gp.name] = args[i]
		}
	}
	return gm, nil
}

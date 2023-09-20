package parsed

import (
	"oak-compiler/pkg/a"
)

func NewTuplePattern(c a.Cursor, items []Pattern) Pattern {
	return patternTuple{patternBase: patternBase{cursor: c}, items: items}
}

definedType patternTuple struct {
	patternBase
	items []Pattern
}

func (p patternTuple) SetType(cursor a.Cursor, type_ Type) (Pattern, error) {
	p.type_ = a.Just(type_)
	return p, nil
}
func (p patternTuple) populateLocals(type_ Type, locals *LocalVars, typeVars TypeVars, md *Metadata) error {
	dt, err := type_.dereference(typeVars, md)
	if err != nil {
		return err
	}

	tupleType, ok := dt.(typeTuple)
	if !ok {
		return a.NewError(p.cursor, "expected tuple definedType here, got %s", type_)
	}

	if len(tupleType.items) != len(p.items) {
		return a.NewError(p.cursor, "expected %d-tuple got %d-tuple", len(tupleType.items), len(p.items))
	}

	for i, itemType := range tupleType.items {
		err = p.items[i].populateLocals(itemType, locals, typeVars, md)
		if err != nil {
			return err
		}
	}
	return nil
}

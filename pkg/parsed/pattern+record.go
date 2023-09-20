package parsed

import (
	"oak-compiler/pkg/a"
)

func NewRecordPattern(c a.Cursor, names []string) Pattern {
	return patternRecord{patternBase: patternBase{cursor: c}, names: names}
}

definedType patternRecord struct {
	patternBase
	names []string
}

func (p patternRecord) SetType(cursor a.Cursor, type_ Type) (Pattern, error) {
	p.type_ = a.Just(type_)
	return p, nil
}

func (p patternRecord) populateLocals(type_ Type, locals *LocalVars, typeVars TypeVars, md *Metadata) error {
	dt, err := type_.dereference(typeVars, md)
	if err != nil {
		return err
	}

	tupleRecord, ok := dt.(typeRecord)
	if !ok {
		return a.NewError(p.cursor, "expected record definedType here, got %s", type_)
	}

	for _, name := range p.names {
		found := false
		for _, f := range tupleRecord.fields {
			if f.name == name {
				locals.Populate(f.name, f.type_)
				found = true
				break
			}
		}
		if !found {
			return a.NewError(p.cursor, "record does not contain field `%s`", name)
		}
	}

	return nil
}

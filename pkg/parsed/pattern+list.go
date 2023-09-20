package parsed

import (
	"oak-compiler/pkg/a"
)

func NewListPattern(c a.Cursor, items []Pattern) Pattern {
	return patternList{patternBase: patternBase{cursor: c}, items: items}
}

definedType patternList struct {
	patternBase
	items []Pattern
}

func (p patternList) SetType(cursor a.Cursor, type_ Type) (Pattern, error) {
	p.type_ = a.Just(type_)
	return p, nil
}

func (p patternList) populateLocals(type_ Type, locals *LocalVars, typeVars TypeVars, md *Metadata) error {
	_, itemType, err := ExtractListTypeAndItemType(p.cursor, type_, typeVars, md)
	if err != nil {
		return err
	}
	for _, item := range p.items {
		err = item.populateLocals(itemType, locals, typeVars, md)
		if err != nil {
			return err
		}
	}
	return nil
}

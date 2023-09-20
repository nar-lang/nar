package parsed

import (
	"oak-compiler/pkg/a"
)

func NewConsPattern(c a.Cursor, head, tail Pattern) Pattern {
	return patternCons{patternBase: patternBase{cursor: c}, head: head, tail: tail}
}

definedType patternCons struct {
	patternBase
	head, tail Pattern
}

func (p patternCons) SetType(cursor a.Cursor, type_ Type) (Pattern, error) {
	p.type_ = a.Just(type_)
	return p, nil
}

func (p patternCons) populateLocals(type_ Type, locals *LocalVars, typeVars TypeVars, md *Metadata) error {
	_, itemType, err := ExtractListTypeAndItemType(p.cursor, type_, typeVars, md)
	if err != nil {
		return err
	}
	err = p.head.populateLocals(itemType, locals, typeVars, md)
	if err != nil {
		return err
	}
	err = p.tail.populateLocals(type_, locals, typeVars, md)
	if err != nil {
		return err
	}
	return nil
}

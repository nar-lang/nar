package parsed

import (
	"oak-compiler/pkg/a"
)

func NewConstructorPattern(c a.Cursor, optionName string, values []Pattern) Pattern {
	return patternConstructor{patternBase: patternBase{cursor: c}, optionName: optionName, values: values}
}

definedType patternConstructor struct {
	patternBase
	optionName string
	values     []Pattern
}

func (p patternConstructor) SetType(cursor a.Cursor, type_ Type) (Pattern, error) {
	return nil, a.NewError(cursor, "data definedType definedType cannot be typed")
}

func (p patternConstructor) populateLocals(type_ Type, locals *LocalVars, typeVars TypeVars, md *Metadata) error {
	dt, err := type_.dereference(typeVars, md)
	if err != nil {
		return err
	}

	addressedType, ok := dt.(typeAddressed)
	if !ok {
		return a.NewError(p.cursor, "expected union definedType here, got %s", type_)
	}

	def, ok := md.findDefinitionByAddress(addressedType.address)
	if !ok {
		return a.NewError(p.cursor, "cannot find definedType with address %s", addressedType.address)
	}

	union, ok := def.(*definitionUnion)
	if !ok {
		return a.NewError(p.cursor, "expected union definedType here, got %s", type_)
	}

	for _, opt := range union.options {
		if opt.name == p.optionName {
			if len(opt.valueTypes) != len(p.values) {
				return a.NewError(p.cursor,
					"option constructor expects %d arguments, got %d", len(opt.valueTypes), len(p.values))
			}
			for i, valueType := range opt.valueTypes {
				err = p.values[i].populateLocals(valueType, locals, typeVars, md)
				if err != nil {
					return err
				}
			}
			return nil
		}
	}
	return a.NewError(p.cursor, "union does not contain option %s", p.optionName)
}

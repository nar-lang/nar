package parsed

import (
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/resolved"
)

func NewInfixDefinition(
	c misc.Cursor, address DefinitionAddress, hidden bool, assoc InfixAssociativity, priority int, alias string,
) Definition {
	return definitionInfix{
		definitionBase: definitionBase{
			Address: address,
			Hidden:  hidden,
			cursor:  c,
		},
		Associativity: assoc,
		Priority:      priority,
		Alias:         alias,
	}
}

type definitionInfix struct {
	DefinitionInfix__ int
	definitionBase
	Associativity InfixAssociativity
	Priority      int
	Alias         string
}

func (def definitionInfix) precondition(*Metadata) (Definition, error) {
	return def, nil
}

func (def definitionInfix) getType(cursor misc.Cursor, generics GenericArgs, md *Metadata) (Type, GenericArgs, error) {
	return typeInfix{definition: def}, def.GenericParams.toArgs(), nil
}

func (def definitionInfix) nestedDefinitionNames() []string {
	return nil
}

func (def definitionInfix) unpackNestedDefinitions() []Definition {
	return nil
}

func (def definitionInfix) resolveName(cursor misc.Cursor, md *Metadata) (string, error) {
	addr := def.Address
	addr.definitionName = def.Alias
	fn, ok := md.findDefinitionByAddress(addr)
	if !ok {
		return "", misc.NewError(
			cursor, "cannot find `%s` infix function alias `%s`", def.Name(), addr.definitionName,
		)
	}
	return fn.resolveName(cursor, md)
}

func (def definitionInfix) resolve(md *Metadata) (resolved.Definition, bool, error) {
	return nil, false, nil
}

type InfixAssociativity string

const (
	InfixAssociativityLeft  InfixAssociativity = "left"
	InfixAssociativityRight                    = "right"
	InfixAssociativityNon                      = "non"
)

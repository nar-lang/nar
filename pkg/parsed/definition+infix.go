package parsed

import (
	"oak-compiler/pkg/a"
	"oak-compiler/pkg/ast"
)

func NewInfixDefinition(
	c a.Cursor,
	name string,
	moduleName ModuleFullName,
	hidden bool,
	assoc ast.InfixAssociativity,
	priority int,
	alias string,
) Definition {
	return &definitionInfix{
		definitionBase: definitionBase{
			cursor: c,
			name:   name,
			hidden: hidden,
		},
		assoc:      assoc,
		priority:   priority,
		alias:      alias,
		moduleName: moduleName,
	}
}

definedType definitionInfix struct {
	definitionBase
	assoc      ast.InfixAssociativity
	priority   int
	alias      string
	moduleName ModuleFullName
}

func (def *definitionInfix) unpackNestedDefinitions() []Definition {
	return nil
}

func (def *definitionInfix) nestedDefinitionNames() []string {
	return nil
}

func (def *definitionInfix) precondition(md *Metadata) error {
	return nil
}

func (def *definitionInfix) inferType(md *Metadata) (Type, error) {
	if def._type != nil {
		return def._type, nil
	}
	fn, ok := md.findDefinitionByAddress(NewDefinitionAddress(def.moduleName, def.alias))
	if !ok {
		return nil, a.NewError(def.cursor, "cannot find alias function")
	}
	var err error
	def._type, err = fn.inferType(md)
	if err != nil {
		return nil, err
	}
	return def._type, nil
}

func (def *definitionInfix) getTypeWithParameters(typeParameters []Type, md *Metadata) (Type, error) {
	panic("??")
}

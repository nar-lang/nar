package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

type TRecord struct {
	*typeBase
	fields map[ast.Identifier]Type
}

func NewTRecord(loc ast.Location, fields map[ast.Identifier]Type) Type {
	return &TRecord{
		typeBase: newTypeBase(loc),
		fields:   fields,
	}
}

func (t *TRecord) Fields() map[ast.Identifier]Type {
	return t.fields
}

func (t *TRecord) normalize(
	modules map[ast.QualifiedIdentifier]*Module, module *Module, typeModule *Module, namedTypes namedTypeMap,
) (normalized.Type, error) {
	fields := map[ast.Identifier]normalized.Type{}
	for n, v := range t.fields {
		var err error
		fields[n], err = v.normalize(modules, module, typeModule, namedTypes)
		if err != nil {
			return nil, err
		}
	}
	return t.setSuccessor(normalized.NewTRecord(t.location, fields))
}

package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

func NewTRecord(loc ast.Location, fields map[ast.Identifier]Type) Type {
	return &TRecord{
		typeBase: newTypeBase(loc),
		fields:   fields,
	}
}

type TRecord struct {
	*typeBase
	fields map[ast.Identifier]Type
}

func (t *TRecord) SemanticTokens() []ast.SemanticToken {
	return []ast.SemanticToken{t.location.ToToken(ast.TokenTypeStruct)}
}

func (t *TRecord) Iterate(f func(statement Statement)) {
	f(t)
	for _, field := range t.fields {
		if field != nil {
			field.Iterate(f)
		}
	}
}

func (t *TRecord) Fields() map[ast.Identifier]Type {
	return t.fields
}

func (t *TRecord) normalize(modules map[ast.QualifiedIdentifier]*Module, module *Module, namedTypes namedTypeMap) (normalized.Type, error) {
	fields := map[ast.Identifier]normalized.Type{}
	for n, v := range t.fields {
		var err error
		fields[n], err = v.normalize(modules, module, namedTypes)
		if err != nil {
			return nil, err
		}
	}
	return t.setSuccessor(normalized.NewTRecord(t.location, fields))
}

func (t *TRecord) applyArgs(params map[ast.Identifier]Type, loc ast.Location) (Type, error) {
	var err error
	fields := map[ast.Identifier]Type{}
	for name, f := range t.fields {
		fields[name], err = f.applyArgs(params, loc)
		if err != nil {
			return nil, err
		}
	}
	return NewTRecord(loc, fields), nil
}

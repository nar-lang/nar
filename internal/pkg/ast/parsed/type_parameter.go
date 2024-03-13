package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
)

func NewTParameter(loc ast.Location, name ast.Identifier) Type {
	return &TParameter{
		typeBase: newTypeBase(loc),
		name:     name,
	}
}

type TParameter struct {
	*typeBase
	name ast.Identifier
}

func (t *TParameter) SemanticTokens() []ast.SemanticToken {
	return []ast.SemanticToken{t.location.ToToken(ast.TokenTypeTypeParameter)}
}

func (t *TParameter) Iterate(f func(statement Statement)) {
	f(t)
}

func (t *TParameter) normalize(modules map[ast.QualifiedIdentifier]*Module, module *Module, namedTypes namedTypeMap) (normalized.Type, error) {
	return t.setSuccessor(normalized.NewTParameter(t.location, t.name))
}

func (t *TParameter) applyArgs(params map[ast.Identifier]Type, loc ast.Location) (Type, error) {
	if p, ok := params[t.name]; !ok || p == nil {
		return nil, common.NewErrorAt(t.location, "missing type parameter %s", t.name)
	} else {
		return p, nil
	}
}

package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

func NewPAny(loc ast.Location) Pattern {
	return &PAny{
		patternBase: newPatternBase(loc),
	}
}

type PAny struct {
	*patternBase
}

func (e *PAny) SemanticTokens() []ast.SemanticToken {
	return []ast.SemanticToken{e.location.ToToken(ast.TokenTypeVariable, ast.TokenModifierDeclaration)}
}

func (e *PAny) Iterate(f func(statement Statement)) {
	f(e)
	e.patternBase.Iterate(f)
}

func (e *PAny) normalize(
	locals map[ast.Identifier]normalized.Pattern,
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	normalizedModule *normalized.Module,
) (normalized.Pattern, error) {
	var declaredType normalized.Type
	var err error
	if e.declaredType != nil {
		declaredType, err = e.declaredType.normalize(modules, module, nil)
	}
	return e.setSuccessor(normalized.NewPAny(e.location, declaredType)), err
}

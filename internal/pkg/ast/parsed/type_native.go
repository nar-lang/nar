package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
)

func NewTNative(loc ast.Location, name ast.FullIdentifier, args []Type, nameLocation ast.Location) Type {
	return &TNative{
		typeBase:     newTypeBase(loc),
		name:         name,
		args:         args,
		nameLocation: nameLocation,
	}
}

type TNative struct {
	*typeBase
	name         ast.FullIdentifier
	args         []Type
	nameLocation ast.Location
}

func (t *TNative) SemanticTokens() []ast.SemanticToken {
	return []ast.SemanticToken{t.nameLocation.ToToken(ast.TokenTypeInterface)}
}

func (t *TNative) Iterate(f func(statement Statement)) {
	f(t)
	for _, arg := range t.args {
		if arg != nil {
			arg.Iterate(f)
		}
	}
}

func (t *TNative) normalize(modules map[ast.QualifiedIdentifier]*Module, module *Module, namedTypes namedTypeMap) (normalized.Type, error) {
	var args []normalized.Type
	for _, arg := range t.args {
		nArg, err := arg.normalize(modules, module, namedTypes)
		if err != nil {
			return nil, err
		}
		args = append(args, nArg)
	}
	return t.setSuccessor(normalized.NewTNative(t.location, t.name, args))
}

func (t *TNative) applyArgs(params map[ast.Identifier]Type, loc ast.Location) (Type, error) {
	var args []Type
	for _, arg := range t.args {
		nArg, err := arg.applyArgs(params, loc)
		if err != nil {
			return nil, err
		}
		args = append(args, nArg)
	}
	return NewTNative(loc, t.name, args, t.nameLocation), nil
}

package parsed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
	"strings"
)

func NewTNamed(loc ast.Location, name ast.QualifiedIdentifier, args []Type) Type {
	return &TNamed{
		typeBase: newTypeBase(loc),
		name:     name,
		args:     args,
	}
}

type TNamed struct {
	*typeBase
	name ast.QualifiedIdentifier
	args []Type
}

func (t *TNamed) Iterate(f func(statement Statement)) {
	f(t)
	for _, arg := range t.args {
		if arg != nil {
			arg.Iterate(f)
		}
	}
}

func (t *TNamed) Find(
	modules map[ast.QualifiedIdentifier]*Module, module *Module,
) (Type, *Module, []ast.FullIdentifier, error) {
	return module.findType(modules, t.name, t.args, t.location)
}

func (t *TNamed) normalize(modules map[ast.QualifiedIdentifier]*Module, module *Module, namedTypes namedTypeMap) (normalized.Type, error) {
	x, _, ids, err := t.Find(modules, module)
	if err != nil {
		return nil, err
	}
	if ids == nil {
		args := ""
		if len(t.args) > 0 {
			args = fmt.Sprintf("[%s]", strings.Join(common.Repeat("_", len(t.args)), ", "))
		}
		return nil, common.Error{Location: t.location, Message: fmt.Sprintf("type `%s%s` not found", t.name, args)}
	}
	if len(ids) > 1 {
		return nil, common.Error{
			Location: t.location,
			Message: fmt.Sprintf(
				"ambiguous type `%s`, it can be one of %s. Use import or qualified name to clarify which one to use",
				t.name, common.Join(ids, ", ")),
		}
	}
	if named, ok := x.(*TNamed); ok {
		if named.name == t.name {
			return nil, common.Error{
				Location: named.location,
				Message:  fmt.Sprintf("type `%s` aliased to itself", t.name),
			}
		}
	}

	nType, err := x.normalize(modules, module, namedTypes)
	if err != nil {
		return nil, err
	}
	return t.setSuccessor(nType)
}

func (t *TNamed) applyArgs(params map[ast.Identifier]Type, loc ast.Location) (Type, error) {
	var args []Type
	for _, arg := range t.args {
		nArg, err := arg.applyArgs(params, loc)
		if err != nil {
			return nil, err
		}
		args = append(args, nArg)
	}
	return NewTNamed(loc, t.name, args), nil
}

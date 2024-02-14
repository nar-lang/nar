package parsed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
	"strings"
)

type TNamed struct {
	*typeBase
	name ast.QualifiedIdentifier
	args []Type
}

func NewTNamed(loc ast.Location, name ast.QualifiedIdentifier, args []Type) Type {
	return &TNamed{
		typeBase: newTypeBase(loc),
		name:     name,
		args:     args,
	}
}

func (t *TNamed) Find(
	modules map[ast.QualifiedIdentifier]*Module, module *Module,
) (Type, *Module, []ast.FullIdentifier, error) {
	return findParsedType(modules, module, t.name, t.args, t.location)
}

func (t *TNamed) normalize(
	modules map[ast.QualifiedIdentifier]*Module, module *Module, typeModule *Module, namedTypes namedTypeMap,
) (normalized.Type, error) {
	x, m, ids, err := t.Find(modules, module)
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

	nType, err := x.normalize(modules, module, m, namedTypes)
	if err != nil {
		return nil, err
	}
	return t.setSuccessor(nType)
}

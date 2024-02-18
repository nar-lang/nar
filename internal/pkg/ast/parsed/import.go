package parsed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
	"slices"
	"strings"
)

type Import interface {
	_parsed()
	exposes(name string) bool
	Module() ast.QualifiedIdentifier
	unwrap(modules map[ast.QualifiedIdentifier]*Module) error
	Alias() *ast.Identifier
}

func NewImport(
	loc ast.Location, module ast.QualifiedIdentifier, alias *ast.Identifier, exposingAll bool, exposing []string,
) Import {
	return &import_{
		location:    loc,
		module_:     module,
		alias:       alias,
		exposingAll: exposingAll,
		exposing:    exposing,
	}
}

type import_ struct {
	location    ast.Location
	module_     ast.QualifiedIdentifier
	alias       *ast.Identifier
	exposingAll bool
	exposing    []string
}

func (i *import_) Alias() *ast.Identifier {
	return i.alias
}

func (i *import_) Module() ast.QualifiedIdentifier {
	return i.module_
}

func (i *import_) exposes(name string) bool {
	return slices.Contains(i.exposing, name)
}

func (i *import_) _parsed() {}

func (imp *import_) unwrap(modules map[ast.QualifiedIdentifier]*Module) error {
	m, ok := modules[imp.module_]
	if !ok {
		return common.NewErrorAt(imp.location, "module `%s` not found", imp.module_)
	}
	modName := m.name
	if imp.alias != nil {
		modName = ast.QualifiedIdentifier(*imp.alias)
	}
	shortModName := ast.QualifiedIdentifier("")
	lastDotIndex := strings.LastIndex(string(modName), ".")
	if lastDotIndex >= 0 {
		shortModName = modName[lastDotIndex+1:]
	}

	var exp []string
	expose := func(n string, exn string) {
		if imp.exposingAll || slices.Contains(imp.exposing, exn) {
			exp = append(exp, n)
		}
		exp = append(exp, fmt.Sprintf("%s.%s", modName, n))
		if shortModName != "" {
			exp = append(exp, fmt.Sprintf("%s.%s", shortModName, n))
		}
	}

	for _, d := range m.definitions {
		if !d.Hidden() {
			expose(string(d.Name()), string(d.Name()))
		}
	}

	for _, a := range m.aliases {
		if !a.Hidden() {
			expose(string(a.Name()), string(a.Name()))
			if dt, ok := a.aliasType().(*TData); ok {
				for _, v := range dt.options {
					if !v.hidden {
						expose(string(v.name), string(a.Name()))
					}
				}
			}
		}
	}

	for _, a := range m.infixFns {
		if !a.hidden() {
			expose(string(a.name()), string(a.name()))
		}
	}
	imp.exposing = exp
	return nil
}

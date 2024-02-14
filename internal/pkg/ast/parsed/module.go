package parsed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
	"slices"
	"strings"
)

type Import struct {
	location         ast.Location
	moduleIdentifier ast.QualifiedIdentifier
	alias            *ast.Identifier
	exposingAll      bool
	exposing         []string
}

func NewImport(
	loc ast.Location, module ast.QualifiedIdentifier, alias *ast.Identifier, exposingAll bool, exposing []string,
) *Import {
	return &Import{
		location:         loc,
		moduleIdentifier: module,
		alias:            alias,
		exposingAll:      exposingAll,
		exposing:         exposing,
	}
}

type Alias struct {
	location ast.Location
	hidden   bool
	name     ast.Identifier
	params   []ast.Identifier
	type_    Type
}

func NewAlias(loc ast.Location, hidden bool, name ast.Identifier, params []ast.Identifier, type_ Type) *Alias {
	return &Alias{
		location: loc,
		hidden:   hidden,
		name:     name,
		params:   params,
		type_:    type_,
	}
}

type Associativity int

const (
	Left  Associativity = -1
	None                = 0
	Right               = 1
)

type Infix struct {
	location      ast.Location
	hidden        bool
	name          ast.InfixIdentifier
	associativity Associativity
	precedence    int
	aliasLocation ast.Location
	alias         ast.Identifier
}

func NewInfix(
	loc ast.Location, hidden bool, name ast.InfixIdentifier, associativity Associativity,
	precedence int, aliasLoc ast.Location, alias ast.Identifier,
) *Infix {
	return &Infix{
		location:      loc,
		hidden:        hidden,
		name:          name,
		associativity: associativity,
		precedence:    precedence,
		aliasLocation: aliasLoc,
		alias:         alias,
	}
}

type DataTypeOption struct {
	location ast.Location
	hidden   bool
	name     ast.Identifier
	values   []Type
}

func NewDataTypeOption(loc ast.Location, hidden bool, name ast.Identifier, values []Type) *DataTypeOption {
	return &DataTypeOption{
		location: loc,
		hidden:   hidden,
		name:     name,
		values:   values,
	}
}

type DataType struct {
	location ast.Location
	hidden   bool
	name     ast.Identifier
	params   []ast.Identifier
	options  []*DataTypeOption
}

func NewDataType(
	loc ast.Location, hidden bool, name ast.Identifier, params []ast.Identifier, options []*DataTypeOption,
) *DataType {
	return &DataType{
		location: loc,
		hidden:   hidden,
		name:     name,
		params:   params,
		options:  options,
	}
}

type Module struct {
	name        ast.QualifiedIdentifier
	location    ast.Location
	imports     []*Import
	aliases     []*Alias
	infixFns    []*Infix
	definitions []*Definition
	dataTypes   []*DataType

	packageName        ast.PackageIdentifier
	referencedPackages map[ast.PackageIdentifier]struct{}
}

func NewModule(
	name ast.QualifiedIdentifier, loc ast.Location,
	imports []*Import, aliases []*Alias, infixFns []*Infix, definitions []*Definition, dataTypes []*DataType,
) *Module {
	return &Module{
		name:               name,
		location:           loc,
		imports:            imports,
		aliases:            aliases,
		infixFns:           infixFns,
		definitions:        definitions,
		dataTypes:          dataTypes,
		referencedPackages: map[ast.PackageIdentifier]struct{}{},
	}
}

func (module *Module) Name() ast.QualifiedIdentifier {
	return module.name
}

func (module *Module) GetLocation() ast.Location {
	return module.location
}

func (module *Module) PackageName() ast.PackageIdentifier {
	return module.packageName
}

func (module *Module) SetPackageName(packageName ast.PackageIdentifier) {
	module.packageName = packageName
}

func (module *Module) SetReferencedPackages(referencedPackages map[ast.PackageIdentifier]struct{}) {
	module.referencedPackages = referencedPackages
}

func (module *Module) PreNormalize(
	modules map[ast.QualifiedIdentifier]*Module,
) (errors []error) {
	module.flattenDataTypes()
	return module.unwrapImports(modules)
}

func (module *Module) Normalize(
	modules map[ast.QualifiedIdentifier]*Module,
	normalizedModules map[ast.QualifiedIdentifier]*normalized.Module,
) (errors []error) {
	if _, ok := normalizedModules[module.name]; ok {
		return
	}

	o := normalized.NewModule(module.location, module.name, nil)

	for _, def := range module.definitions {
		nDef, params, err := def.normalize(modules, module, o)
		if err != nil {
			errors = append(errors, err)
		}
		nDef.FlattenLambdas(params, o)

		o.AddDefinition(nDef)
	}

	normalizedModules[module.name] = o

	for _, modName := range o.Dependencies() {
		depModule, ok := modules[modName]
		if !ok {
			errors = append(errors,
				common.Error{Location: depModule.location, Message: fmt.Sprintf("module `%s` not found", modName)},
			)
			continue
		}

		if err := depModule.Normalize(modules, normalizedModules); err != nil {
			errors = append(errors, err...)
		}
	}

	return
}

func (module *Module) flattenDataTypes() {
	for _, it := range module.dataTypes {
		typeArgs := common.Map(func(x ast.Identifier) Type {
			return NewTParameter(it.location, x)
		}, it.params)

		dataType := NewTData(
			it.location,
			common.MakeFullIdentifier(module.name, it.name),
			typeArgs,
			common.Map(func(x *DataTypeOption) DataOption {
				return NewDataOption(x.name, x.hidden, x.values)
			}, it.options),
		)
		module.aliases = append(module.aliases, NewAlias(it.location, it.hidden, it.name, it.params, dataType))
		for _, option := range it.options {
			type_ := dataType
			if len(option.values) > 0 {
				type_ = NewTFunc(it.location, option.values, type_)
			}
			body := NewConstructor(
				option.location,
				module.name,
				it.name,
				option.name,
				common.Map(
					func(i int) Expression {
						return NewVar(option.location, ast.QualifiedIdentifier(fmt.Sprintf("p%d", i)))
					},
					common.Range(0, len(option.values)),
				))

			params := common.Map(
				func(i int) Pattern {
					return NewPNamed(option.location, ast.Identifier(fmt.Sprintf("p%d", i)))
				},
				common.Range(0, len(option.values)),
			)

			module.definitions = append(module.definitions,
				NewDefinition(option.location, option.hidden || it.hidden, option.name, params, body, type_))
		}
	}
}

func (module *Module) unwrapImports(modules map[ast.QualifiedIdentifier]*Module) (errors []error) {
	for i, imp := range module.imports {
		m, ok := modules[imp.moduleIdentifier]
		if !ok {
			errors = append(errors, common.Error{
				Location: imp.location,
				Message:  fmt.Sprintf("module `%s` not found", imp.moduleIdentifier),
			})
			continue
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
			if !d.hidden {
				expose(string(d.name), string(d.name))
			}
		}

		for _, a := range m.aliases {
			if !a.hidden {
				expose(string(a.name), string(a.name))
				if dt, ok := a.type_.(*TData); ok {
					for _, v := range dt.options {
						if !v.hidden {
							expose(string(v.name), string(a.name))
						}
					}
				}
			}
		}

		for _, a := range m.infixFns {
			if !a.hidden {
				expose(string(a.name), string(a.name))
			}
		}
		imp.exposing = exp
		module.imports[i] = imp
	}
	return
}

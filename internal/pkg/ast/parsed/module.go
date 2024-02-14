package parsed

import (
	"nar-compiler/internal/pkg/ast"
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

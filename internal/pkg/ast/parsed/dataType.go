package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
)

type DataType interface {
	Statement
	flatten(name ast.QualifiedIdentifier) (Alias, []Definition)
	Name() ast.Identifier
	Options() []DataTypeOption
	Hidden() bool
}

func NewDataType(
	loc ast.Location, hidden bool, name ast.Identifier, params []ast.Identifier, options []DataTypeOption,
	nameLocation ast.Location,
) DataType {
	return &dataType{
		location:     loc,
		hidden:       hidden,
		name:         name,
		params:       params,
		options:      options,
		nameLocation: nameLocation,
	}
}

type dataType struct {
	location     ast.Location
	hidden       bool
	name         ast.Identifier
	params       []ast.Identifier
	options      []DataTypeOption
	successor    Statement
	nameLocation ast.Location
}

func (d dataType) SemanticTokens() []ast.SemanticToken {
	return []ast.SemanticToken{d.nameLocation.ToToken(ast.TokenTypeEnum, ast.TokenModifierDeclaration)}
}

func (d dataType) Hidden() bool {
	return d.hidden
}

func (d dataType) Options() []DataTypeOption {
	return d.options
}

func (d dataType) Name() ast.Identifier {
	return d.name
}

func (d dataType) Successor() normalized.Statement {
	return d.successor.Successor()
}

func (d dataType) flatten(moduleName ast.QualifiedIdentifier) (Alias, []Definition) {
	typeArgs := common.Map(func(x ast.Identifier) Type {
		return NewTParameter(d.location, x)
	}, d.params)
	type_ := NewTData(
		d.location,
		common.MakeFullIdentifier(moduleName, d.name),
		typeArgs,
		common.Map(func(x DataTypeOption) *DataOption {
			return x.dataOption()
		}, d.options),
		d.nameLocation,
	)
	dataAlias := NewAlias(d.location, d.hidden, d.name, d.params, type_, d.nameLocation)
	defs := make([]Definition, 0, len(d.options))
	for _, option := range d.options {
		def := option.constructor(moduleName, d.name, type_, d.hidden)
		defs = append(defs, def)
	}

	d.successor = dataAlias

	return dataAlias, defs
}

func (d dataType) Location() ast.Location {
	return d.location
}

func (d dataType) Iterate(f func(statement Statement)) {
	f(d)
	for _, option := range d.options {
		option.Iterate(f)
	}
}

func (d dataType) _parsed() {}

type DataTypeOption interface {
	Statement
	dataOption() *DataOption
	constructor(moduleName ast.QualifiedIdentifier, dataName ast.Identifier, dataType Type, hidden bool) Definition
	Name() ast.Identifier
	Hidden() bool
}

func NewDataTypeOption(loc ast.Location, hidden bool, name ast.Identifier, values []*DataTypeValue, nameLocation ast.Location) DataTypeOption {
	return &dataTypeOption{
		location:     loc,
		hidden:       hidden,
		name:         name,
		values:       values,
		nameLocation: nameLocation,
	}
}

type dataTypeOption struct {
	location     ast.Location
	hidden       bool
	name         ast.Identifier
	values       []*DataTypeValue
	successor    Statement
	nameLocation ast.Location
}

func (d *dataTypeOption) SemanticTokens() []ast.SemanticToken {
	return []ast.SemanticToken{d.nameLocation.ToToken(ast.TokenTypeEnumMember, ast.TokenModifierDeclaration)}
}

func (d *dataTypeOption) Hidden() bool {
	return d.hidden
}

func (d *dataTypeOption) Name() ast.Identifier {
	return d.name
}

func (d *dataTypeOption) constructor(moduleName ast.QualifiedIdentifier, dataName ast.Identifier, dataType Type, hidden bool) Definition {
	type_ := dataType
	if len(d.values) > 0 {
		type_ = NewTFunc(d.location, common.Map(func(v *DataTypeValue) Type { return v.type_ }, d.values), type_)
	}
	body := NewConstructor(
		d.location,
		moduleName,
		dataName,
		d.name,
		d.nameLocation,
		common.Map(
			func(i *DataTypeValue) Expression {
				return NewVar(d.location, ast.QualifiedIdentifier("_"+i.name))
			},
			d.values),
	)

	params := common.Map(
		func(i *DataTypeValue) Pattern {
			return NewPNamed(d.location, "_"+i.name, i.location)
		},
		d.values,
	)

	def := NewDefinition(d.location, d.hidden || hidden, d.name, d.location, params, body, type_)
	d.successor = def
	return def
}

func (d *dataTypeOption) dataOption() *DataOption {
	return NewDataOption(d.name, d.hidden, common.Map(func(v *DataTypeValue) Type { return v.type_ }, d.values), d.nameLocation)
}

func (d *dataTypeOption) Successor() normalized.Statement {
	return d.successor.Successor()
}

func (d *dataTypeOption) Location() ast.Location {
	return d.location
}

func (d *dataTypeOption) Iterate(f func(statement Statement)) {
	f(d)
	for _, value := range d.values {
		if value != nil && value.type_ != nil {
			value.type_.Iterate(f)
		}
	}
}

func (d *dataTypeOption) _parsed() {}

type DataTypeValue struct {
	location ast.Location
	name     ast.Identifier
	type_    Type
}

func NewDataTypeValue(loc ast.Location, name ast.Identifier, type_ Type) *DataTypeValue {
	return &DataTypeValue{
		location: loc,
		name:     name,
		type_:    type_,
	}
}

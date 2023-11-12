package typed

import (
	"fmt"
	"oak-compiler/ast"
	"os"
	"path/filepath"
)

type Expression interface {
	fmt.Stringer
	_expression()
	GetType() Type
	GetLocation() ast.Location
}

type Access struct {
	ast.Location
	Type
	FieldName ast.Identifier
	Record    Expression
}

func (e *Access) _expression() {}

func (e *Access) String() string {
	return fmt.Sprintf("Access(%s,%v)", e.FieldName, e.Record)
}

func (e *Access) GetLocation() ast.Location {
	return e.Location
}

func (e *Access) GetType() Type {
	return e.Type
}

type Apply struct {
	ast.Location
	Type
	Func Expression
	Args []Expression
}

func (e *Apply) _expression() {}

func (e *Apply) String() string {
	return fmt.Sprintf("Apply(%v,%v)", e.Func, e.Args)
}

func (e *Apply) GetLocation() ast.Location {
	return e.Location
}

func (e *Apply) GetType() Type {
	return e.Type
}

type Const struct {
	ast.Location
	Type
	Value ast.ConstValue
}

func (e *Const) _expression() {}

func (e *Const) String() string {
	return fmt.Sprintf("Const(%v)", e.Value)
}

func (e *Const) GetLocation() ast.Location {
	return e.Location
}

func (e *Const) GetType() Type {
	return e.Type
}

type If struct {
	ast.Location
	Type
	Condition, Positive, Negative Expression
}

func (e *If) _expression() {}

func (e *If) String() string {
	return fmt.Sprintf("If(%v,%v,%v)", e.Condition, e.Positive, e.Negative)
}

func (e *If) GetLocation() ast.Location {
	return e.Location
}

func (e *If) GetType() Type {
	return e.Type
}

type Let struct {
	ast.Location
	Type
	Definition *Definition
	Body       Expression
}

func (e *Let) _expression() {}

func (e *Let) String() string {
	return fmt.Sprintf("Let(%v,%v)", e.Definition, e.Body)
}

func (e *Let) GetLocation() ast.Location {
	return e.Location
}

func (e *Let) GetType() Type {
	return e.Type
}

type List struct {
	ast.Location
	Type
	Items []Expression
}

func (e *List) _expression() {}

func (e *List) String() string {
	return fmt.Sprintf("List(%v)", e.Items)
}

func (e *List) GetLocation() ast.Location {
	return e.Location
}

func (e *List) GetType() Type {
	return e.Type
}

type RecordField struct {
	ast.Location
	Type
	Name  ast.Identifier
	Value Expression
}

func (rf RecordField) String() string {
	return fmt.Sprintf("%s=%v", rf.Name, rf.Value)
}

type Record struct {
	ast.Location
	Type
	Fields []RecordField
}

func (e *Record) _expression() {}

func (e *Record) String() string {
	return fmt.Sprintf("Record(%v)", e.Fields)
}

func (e *Record) GetLocation() ast.Location {
	return e.Location
}

func (e *Record) GetType() Type {
	return e.Type
}

type SelectCase struct {
	ast.Location
	Type
	Pattern    Pattern
	Expression Expression
}

func (sc SelectCase) String() string {
	return fmt.Sprintf("Case(%s,%s)", sc.Pattern, sc.Expression)
}

type Select struct {
	ast.Location
	Type
	Condition Expression
	Cases     []SelectCase
}

func (e *Select) _expression() {}

func (e *Select) String() string {
	return fmt.Sprintf("Select(%v,%v)", e.Condition, e.Cases)
}

func (e *Select) GetLocation() ast.Location {
	return e.Location
}

func (e *Select) GetType() Type {
	return e.Type
}

type Tuple struct {
	ast.Location
	Type
	Items []Expression
}

func (e *Tuple) _expression() {}

func (e *Tuple) String() string {
	return fmt.Sprintf("Tuple(%v)", e.Items)
}

func (e *Tuple) GetLocation() ast.Location {
	return e.Location
}

func (e *Tuple) GetType() Type {
	return e.Type
}

type UpdateLocal struct {
	ast.Location
	Type
	RecordName ast.Identifier
	Fields     []RecordField
}

func (e *UpdateLocal) _expression() {}

func (e *UpdateLocal) String() string {
	return fmt.Sprintf("UpdateLocal(%v,%v)", e.RecordName, e.Fields)
}

func (e *UpdateLocal) GetLocation() ast.Location {
	return e.Location
}

func (e *UpdateLocal) GetType() Type {
	return e.Type
}

type UpdateGlobal struct {
	ast.Location
	Type
	ModulePath     string
	DefinitionName ast.Identifier
	Definition     *Definition
	Fields         []RecordField
}

func (e *UpdateGlobal) _expression() {}

func (e *UpdateGlobal) String() string {
	return fmt.Sprintf("UpdateGlobal(%s:%s,%v)", e.ModulePath, e.DefinitionName, e.Fields)
}

func (e *UpdateGlobal) GetLocation() ast.Location {
	return e.Location
}

func (e *UpdateGlobal) GetType() Type {
	return e.Type
}

type Lambda struct {
	ast.Location
	Type
	Params []Pattern
	Body   Expression
}

func (e *Lambda) _expression() {}

func (e *Lambda) String() string {
	return fmt.Sprintf("Lambda(%v,%v)", e.Params, e.Body)
}

func (e *Lambda) GetLocation() ast.Location {
	return e.Location
}

func (e *Lambda) GetType() Type {
	return e.Type
}

type Constructor struct {
	ast.Location
	Type
	DataName   ast.ExternalIdentifier
	OptionName ast.DataOptionIdentifier
	Args       []Expression
}

func (e *Constructor) _expression() {}

func (e *Constructor) String() string {
	return fmt.Sprintf("Constructor(%s/%s,%v)", e.DataName, e.OptionName, e.Args)
}

func (e *Constructor) GetLocation() ast.Location {
	return e.Location
}

func (e *Constructor) GetType() Type {
	return e.Type
}

type NativeCall struct {
	ast.Location
	Type
	Name ast.ExternalIdentifier
	Args []Expression
}

func (e *NativeCall) _expression() {}

func (e *NativeCall) String() string {
	return fmt.Sprintf("NativeCall(%s,%v)", e.Name, e.Args)
}

func (e *NativeCall) GetLocation() ast.Location {
	return e.Location
}

func (e *NativeCall) GetType() Type {
	return e.Type
}

type Local struct {
	ast.Location
	Type
	Name ast.Identifier
}

func (e *Local) _expression() {}

func (e *Local) String() string {
	return fmt.Sprintf("Local(%s)", e.Name)
}

func (e *Local) GetLocation() ast.Location {
	return e.Location
}

func (e *Local) GetType() Type {
	return e.Type
}

type Global struct {
	ast.Location
	Type
	ModulePath     string
	DefinitionName ast.Identifier
	Definition     *Definition
}

func (e *Global) _expression() {}

func (e *Global) String() string {
	cwd, _ := os.Getwd()
	p, _ := filepath.Rel(cwd, e.ModulePath)
	return fmt.Sprintf("Global(%s:%s)", p, e.DefinitionName)
}

func (e *Global) GetLocation() ast.Location {
	return e.Location
}

func (e *Global) GetType() Type {
	return e.Type
}

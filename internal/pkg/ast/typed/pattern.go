package typed

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
)

type Pattern interface {
	ToString(m UnboundMap, withType bool, currentModule ast.QualifiedIdentifier) string
	_pattern()
	GetType() Type
	GetLocation() ast.Location
	GetDeclaredType() Type
}

type PAlias struct {
	ast.Location
	Type
	Alias        ast.Identifier
	Nested       Pattern
	DeclaredType Type
}

func (*PAlias) _pattern() {}

func (p *PAlias) ToString(m UnboundMap, withType bool, currentModule ast.QualifiedIdentifier) string {
	s := fmt.Sprintf("%s as %v", p.Nested.ToString(m, false, currentModule), p.Alias)
	if withType {
		s = "(" + s + "): " + p.Type.ToString(m, currentModule)
	}
	return s
}

func (p *PAlias) GetLocation() ast.Location {
	return p.Location
}

func (p *PAlias) GetType() Type {
	return p.Type
}

func (p *PAlias) GetDeclaredType() Type {
	return p.DeclaredType
}

type PAny struct {
	ast.Location
	Type
	DeclaredType Type
}

func (*PAny) _pattern() {}

func (p *PAny) ToString(m UnboundMap, withType bool, currentModule ast.QualifiedIdentifier) string {
	s := "_"
	if withType {
		s += ": " + p.Type.ToString(m, currentModule)
	}
	return s
}

func (p *PAny) GetLocation() ast.Location {
	return p.Location
}

func (p *PAny) GetType() Type {
	return p.Type
}

func (p *PAny) GetDeclaredType() Type {
	return p.DeclaredType
}

type PCons struct {
	ast.Location
	Type
	Head, Tail   Pattern
	DeclaredType Type
}

func (*PCons) _pattern() {}

func (p *PCons) ToString(m UnboundMap, withType bool, currentModule ast.QualifiedIdentifier) string {
	s := p.Head.ToString(m, withType, currentModule) + " | " + p.Tail.ToString(m, false, currentModule)
	if withType {
		s = "(" + s + "): " + p.Type.ToString(m, currentModule)
	}
	return s
}

func (p *PCons) GetLocation() ast.Location {
	return p.Location
}

func (p *PCons) GetType() Type {
	return p.Type
}

func (p *PCons) GetDeclaredType() Type {
	return p.DeclaredType
}

type PConst struct {
	ast.Location
	Type
	Value        ast.ConstValue
	DeclaredType Type
}

func (*PConst) _pattern() {}

func (p *PConst) ToString(m UnboundMap, withType bool, currentModule ast.QualifiedIdentifier) string {
	s := p.Value.String()
	if withType {
		s += ": " + p.Type.ToString(m, currentModule)
	}
	return s
}

func (p *PConst) GetLocation() ast.Location {
	return p.Location
}

func (p *PConst) GetType() Type {
	return p.Type
}

func (p *PConst) GetDeclaredType() Type {
	return p.DeclaredType
}

type PDataOption struct {
	ast.Location
	Type
	DataName     ast.FullIdentifier
	OptionName   ast.Identifier
	Definition   *Definition
	Args         []Pattern
	DeclaredType Type
}

func (*PDataOption) _pattern() {}

func (p *PDataOption) ToString(m UnboundMap, withType bool, currentModule ast.QualifiedIdentifier) string {
	s := string(common.MakeDataOptionIdentifier(p.DataName, p.OptionName))
	if len(p.Args) > 0 {
		s += "(" + common.Fold(func(x Pattern, s string) string {
			if s != "" {
				s += ", "
			}
			return s + x.ToString(m, false, currentModule)
		}, "", p.Args) + ")"
	}
	if withType {
		s += ": " + p.Type.ToString(m, currentModule)
	}
	return s
}

func (p *PDataOption) GetLocation() ast.Location {
	return p.Location
}

func (p *PDataOption) GetType() Type {
	return p.Type
}

func (p *PDataOption) GetDeclaredType() Type {
	return p.DeclaredType
}

type PList struct {
	ast.Location
	Type
	Items        []Pattern
	DeclaredType Type
}

func (*PList) _pattern() {}

func (p *PList) ToString(m UnboundMap, withType bool, currentModule ast.QualifiedIdentifier) string {
	s := "[" + common.Fold(func(x Pattern, s string) string {
		if s != "" {
			s += ", "
		}
		return s + x.ToString(m, false, currentModule)
	}, "", p.Items) + "]"
	if withType {
		s += ": " + p.Type.ToString(m, currentModule)
	}
	return s
}

func (p *PList) GetLocation() ast.Location {
	return p.Location
}

func (p *PList) GetType() Type {
	return p.Type
}

func (p *PList) GetDeclaredType() Type {
	return p.DeclaredType
}

type PNamed struct {
	ast.Location
	Type
	Name         ast.Identifier
	DeclaredType Type
}

func (*PNamed) _pattern() {}

func (p *PNamed) ToString(m UnboundMap, withType bool, currentModule ast.QualifiedIdentifier) string {
	s := string(p.Name)
	if withType {
		s += ": " + p.Type.ToString(m, currentModule)
	}
	return s
}

func (p *PNamed) GetLocation() ast.Location {
	return p.Location
}

func (p *PNamed) GetType() Type {
	return p.Type
}

func (p *PNamed) GetDeclaredType() Type {
	return p.DeclaredType
}

type PRecordField struct {
	ast.Location
	Name         ast.Identifier
	Type         Type
	DeclaredType Type
}

type PRecord struct {
	ast.Location
	Type
	Fields       []PRecordField
	DeclaredType Type
}

func (*PRecord) _pattern() {}

func (p *PRecord) ToString(m UnboundMap, withType bool, currentModule ast.QualifiedIdentifier) string {
	s := "{" + common.Fold(func(x PRecordField, s string) string {
		if s != "" {
			s += ", "
		}
		return s + string(x.Name) + ": " + x.Type.ToString(m, currentModule)
	}, "", p.Fields) + "}"
	return s
}

func (p *PRecord) GetLocation() ast.Location {
	return p.Location
}

func (p *PRecord) GetType() Type {
	return p.Type
}

func (p *PRecord) GetDeclaredType() Type {
	return p.DeclaredType
}

type PTuple struct {
	ast.Location
	Type
	Items        []Pattern
	DeclaredType Type
}

func (*PTuple) _pattern() {}

func (p *PTuple) ToString(m UnboundMap, withType bool, currentModule ast.QualifiedIdentifier) string {
	s := "(" + common.Fold(func(x Pattern, s string) string {
		if s != "" {
			s += ", "
		}
		return s + x.ToString(m, false, currentModule)
	}, "", p.Items) + ")"
	return s
}

func (p *PTuple) GetLocation() ast.Location {
	return p.Location
}

func (p *PTuple) GetType() Type {
	return p.Type
}

func (p *PTuple) GetDeclaredType() Type {
	return p.DeclaredType
}

package ast

type SemanticTokenType int

const (
	TokenTypeNamespace SemanticTokenType = iota
	TokenTypeType
	TokenTypeClass
	TokenTypeEnum
	TokenTypeInterface
	TokenTypeStruct
	TokenTypeTypeParameter
	TokenTypeParameter
	TokenTypeVariable
	TokenTypeProperty
	TokenTypeEnumMember
	TokenTypeEvent
	TokenTypeFunction
	TokenTypeMethod
	TokenTypeMacro
	TokenTypeKeyword
	TokenTypeModifier
	TokenTypeComment
	TokenTypeString
	TokenTypeNumber
	TokenTypeRegexp
	TokenTypeOperator
	TokenTypeDecorator
)

var SemanticTokenTypesLegend = []string{
	"namespace",
	"type",
	"class",
	"enum",
	"interface",
	"struct",
	"typeParameter",
	"parameter",
	"variable",
	"property",
	"enumMember",
	"event",
	"function",
	"method",
	"macro",
	"keyword",
	"modifier",
	"comment",
	"string",
	"number",
	"regexp",
	"operator",
	"decorator",
}

type SemanticTokenModifier int

const (
	TokenModifierDeclaration    SemanticTokenModifier = 0x01
	TokenModifierDefinition                           = 0x02
	TokenModifierReadonly                             = 0x04
	TokenModifierStatic                               = 0x08
	TokenModifierDeprecated                           = 0x10
	TokenModifierAbstract                             = 0x20
	TokenModifierAsync                                = 0x40
	TokenModifierModification                         = 0x80
	TokenModifierDocumentation                        = 0x100
	TokenModifierDefaultLibrary                       = 0x200
)

var SemanticTokenModifiersLegend = []string{
	"declaration",
	"definition",
	"readonly",
	"static",
	"deprecated",
	"abstract",
	"async",
	"modification",
	"documentation",
	"defaultLibrary",
}

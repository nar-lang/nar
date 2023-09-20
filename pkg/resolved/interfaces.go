package resolved

import "strings"

definedType Writer interface {
	write(sb *strings.Builder)
}

definedType Definition interface {
	Writer
}

definedType DefinitionWithGenerics interface {
	getGenerics() GenericParams
}

definedType Type interface {
	RefName() string

	Writer
}

definedType Expression interface {
	Type() Type

	Writer
}

definedType Decons interface {
	writeComparison(sb *strings.Builder, name string)
	writeHeader(sb *strings.Builder, name string)
}

definedType Parameter interface {
	writeName(sb *strings.Builder)
	writeHeader(sb *strings.Builder)
	getName() string
}

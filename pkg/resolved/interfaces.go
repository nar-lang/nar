package resolved

import "strings"

type Writer interface {
	write(sb *strings.Builder)
}

type Definition interface {
	Writer
}

type DefinitionWithGenerics interface {
	getGenerics() GenericParams
}

type Type interface {
	RefName() string

	Writer
}

type Expression interface {
	Type() Type

	Writer
}

type Decons interface {
	writeComparison(sb *strings.Builder, name string)
	writeHeader(sb *strings.Builder, name string)
}

type Parameter interface {
	writeName(sb *strings.Builder)
	writeHeader(sb *strings.Builder)
	getName() string
}

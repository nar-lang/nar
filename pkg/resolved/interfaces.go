package resolved

import "strings"

type Writer interface {
	write(sb *strings.Builder)
}

type Definition interface {
	Writer
}

type DefinitionWithGenerics interface {
	writeGenericsShort(sb *strings.Builder)
	writeGenericsFull(sb *strings.Builder)
}

type Type interface {
	RefName() string

	Writer
}

type Expression interface {
	Writer

	Type() Type
}

type Decons interface {
	writeComparison(sb *strings.Builder, name string)
	writeHeader(sb *strings.Builder, name string)
}

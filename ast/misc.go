package ast

type Identifier string

type QualifiedIdentifier string

type InfixIdentifier string

type ExternalIdentifier string

type Location struct {
	FilePath string
	Position uint64
}

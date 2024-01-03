package ast

import (
	"strings"
)

type Identifier string

type QualifiedIdentifier string

type PackageIdentifier string

type InfixIdentifier string

type FullIdentifier string

func (f FullIdentifier) String() string {
	return string(f)
}

func (f FullIdentifier) Module() QualifiedIdentifier {
	return QualifiedIdentifier(f[:strings.LastIndex(string(f), ".")])
}

type DataOptionIdentifier string

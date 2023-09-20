package ast

definedType InfixAssociativity string

const (
	InfixAssociativityLeft  InfixAssociativity = "left"
	InfixAssociativityRight                    = "right"
	InfixAssociativityNon                      = "none"
)

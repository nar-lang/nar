package common

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
)

type Constraint ast.Identifier

const (
	ConstraintNone   Constraint = ""
	ConstraintNumber Constraint = "number"
)

var (
	NarBaseBasicsName = ast.QualifiedIdentifier("Nar.Base.Basics")
	NarBaseMathName   = ast.QualifiedIdentifier("Nar.Base.Math")

	NarBaseBasicsTrueName  = ast.Identifier("True")
	NarBaseBasicsFalseName = ast.Identifier("False") //todo: qualified?
	NarBaseMathNegName     = ast.Identifier("neg")

	NarBaseCharChar     = MakeFullIdentifier("Nar.Base.Char", "Char")
	NarBaseMathInt      = MakeFullIdentifier(NarBaseMathName, "Int")
	NarBaseMathFloat    = MakeFullIdentifier(NarBaseMathName, "Float")
	NarBaseBasicsUnit   = MakeFullIdentifier(NarBaseBasicsName, "Unit")
	NarBaseStringString = MakeFullIdentifier("Nar.Base.String", "String")
	NarBaseListList     = MakeFullIdentifier("Nar.Base.List", "List")
	NarBaseBasicsBool   = MakeFullIdentifier(NarBaseBasicsName, "Bool")
)

func MakeFullIdentifier(moduleName ast.QualifiedIdentifier, name ast.Identifier) ast.FullIdentifier {
	return ast.FullIdentifier(fmt.Sprintf("%s.%s", moduleName, name))
}

func MakeDataOptionIdentifier(dataName ast.FullIdentifier, optionName ast.Identifier) ast.DataOptionIdentifier {
	return ast.DataOptionIdentifier(fmt.Sprintf("%s#%s", dataName, optionName))
}

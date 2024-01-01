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
	NarCoreBasicsName      = ast.QualifiedIdentifier("Nar.Core.Basics")
	NarCoreBasicsTrueName  = ast.Identifier("True")
	NarCoreBasicsFalseName = ast.Identifier("False")

	NarCoreMath         = ast.QualifiedIdentifier("Nar.Core.Math")
	NarCoreMathNeg      = ast.Identifier("neg")
	NarCoreCharChar     = MakeFullIdentifier("Nar.Core.Char", "Char")
	NarCoreMathInt      = MakeFullIdentifier("Nar.Core.Math", "Int")
	NarCoreMathFloat    = MakeFullIdentifier("Nar.Core.Math", "Float")
	NarCoreBasicsUnit   = MakeFullIdentifier(NarCoreBasicsName, "Unit")
	NarCoreStringString = MakeFullIdentifier("Nar.Core.String", "String")
	NarCoreListList     = MakeFullIdentifier("Nar.Core.List", "List")
	Number              = MakeFullIdentifier("", ast.Identifier(ConstraintNumber))
	NarCoreBasicsBool   = MakeFullIdentifier(NarCoreBasicsName, "Bool")
)

func MakeFullIdentifier(moduleName ast.QualifiedIdentifier, name ast.Identifier) ast.FullIdentifier {
	return ast.FullIdentifier(fmt.Sprintf("%s.%s", moduleName, name))
}

func MakeDataOptionIdentifier(dataName ast.FullIdentifier, optionName ast.Identifier) ast.DataOptionIdentifier {
	return ast.DataOptionIdentifier(fmt.Sprintf("%s#%s", dataName, optionName))
}

package common

import (
	"fmt"
	"oak-compiler/internal/pkg/ast"
)

type Constraint ast.Identifier

const (
	ConstraintNone   Constraint = ""
	ConstraintNumber Constraint = "number"
)

var (
	OakCoreBasicsName      = ast.QualifiedIdentifier("Oak.Core.Basics")
	OakCoreBasicsTrueName  = ast.Identifier("True")
	OakCoreBasicsFalseName = ast.Identifier("False")

	OakCoreMath         = ast.QualifiedIdentifier("Oak.Core.Math")
	OakCoreMathNeg      = ast.Identifier("neg")
	OakCoreCharChar     = MakeFullIdentifier("Oak.Core.Char", "Char")
	OakCoreMathInt      = MakeFullIdentifier("Oak.Core.Math", "Int")
	OakCoreMathFloat    = MakeFullIdentifier("Oak.Core.Math", "Float")
	OakCoreBasicsUnit   = MakeFullIdentifier(OakCoreBasicsName, "Unit")
	OakCoreStringString = MakeFullIdentifier("Oak.Core.String", "String")
	OakCoreListList     = MakeFullIdentifier("Oak.Core.List", "List")
	Number              = MakeFullIdentifier("", ast.Identifier(ConstraintNumber))
	OakCoreBasicsBool   = MakeFullIdentifier(OakCoreBasicsName, "Bool")
)

func MakeFullIdentifier(moduleName ast.QualifiedIdentifier, name ast.Identifier) ast.FullIdentifier {
	return ast.FullIdentifier(fmt.Sprintf("%s.%s", moduleName, name))
}

func MakeDataOptionIdentifier(dataName ast.FullIdentifier, optionName ast.Identifier) ast.DataOptionIdentifier {
	return ast.DataOptionIdentifier(fmt.Sprintf("%s#%s", dataName, optionName))
}

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
	OakCoreCharChar     = MakeExternalIdentifier("Oak.Core.Char", "Char")
	OakCoreBasicsInt    = MakeExternalIdentifier("Oak.Core.Math", "Int")
	OakCoreBasicsFloat  = MakeExternalIdentifier("Oak.Core.Math", "Float")
	OakCoreBasicsUnit   = MakeExternalIdentifier(OakCoreBasicsName, "Unit")
	OakCoreStringString = MakeExternalIdentifier("Oak.Core.String", "String")
	OakCoreListList     = MakeExternalIdentifier("Oak.Core.List", "List")
	Number              = MakeExternalIdentifier("", ast.Identifier(ConstraintNumber))
	OakCoreBasicsBool   = MakeExternalIdentifier(OakCoreBasicsName, "Bool")
)

func MakeExternalIdentifier(moduleName ast.QualifiedIdentifier, name ast.Identifier) ast.ExternalIdentifier {
	return ast.ExternalIdentifier(fmt.Sprintf("%s.%s", moduleName, name))
}

func MakeDataOptionIdentifier(dataName ast.ExternalIdentifier, optionName ast.Identifier) ast.DataOptionIdentifier {
	return ast.DataOptionIdentifier(fmt.Sprintf("%s#%s", dataName, optionName))
}

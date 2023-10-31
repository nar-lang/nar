package common

import (
	"fmt"
	"oak-compiler/ast"
)

var (
	OakCoreBasicsNeg    = MakeExternalIdentifier("Oak.Core.Basics", "neg")
	OakCoreCharChar     = MakeExternalIdentifier("Oak.Core.Char", "Char")
	OakCoreBasicsInt    = MakeExternalIdentifier("Oak.Core.Basics", "Int")
	OakCoreBasicsFloat  = MakeExternalIdentifier("Oak.Core.Basics", "Float")
	OakCoreBasicsUnit   = MakeExternalIdentifier("Oak.Core.Basics", "Unit")
	OakCoreBasicsBool   = MakeExternalIdentifier("Oak.Core.Basics", "Bool")
	OakCoreStringString = MakeExternalIdentifier("Oak.Core.String", "String")
	OakCoreListList     = MakeExternalIdentifier("Oak.Core.List", "List")
)

func MakeExternalIdentifier(moduleName ast.QualifiedIdentifier, name ast.Identifier) ast.ExternalIdentifier {
	return ast.ExternalIdentifier(fmt.Sprintf("%s/%s", moduleName, name))
}

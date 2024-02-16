package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
)

func newAmbiguousInfixError(ids []ast.FullIdentifier, name ast.InfixIdentifier, loc ast.Location) error {
	if len(ids) == 0 {
		return common.NewErrorAt(loc, "infix definition `%s` not found", name)
	} else {
		return common.NewErrorAt(loc,
			"ambiguous infix identifier `%s`, it can be one of %s. "+
				"Use import to clarify which one to use",
			name, common.Join(ids, ", "))
	}
}

func newAmbiguousDefinitionError(ids []ast.FullIdentifier, name ast.QualifiedIdentifier, loc ast.Location) error {
	if len(ids) == 0 {
		return common.NewErrorAt(loc, "definition `%s` not found", name)
	} else {
		return common.NewErrorAt(loc,
			"ambiguous identifier `%s`, it can be one of %s. "+
				"Use import or qualified identifer to clarify which one to use",
			name, common.Join(ids, ", "))
	}
}

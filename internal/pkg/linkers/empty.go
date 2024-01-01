package linkers

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
)

type EmptyLinker struct {
}

func (l EmptyLinker) GetOutFileLocation(givenLocation string) string {
	return givenLocation + ".acorn"
}

func (l EmptyLinker) Link(
	main ast.FullIdentifier, packages map[ast.PackageIdentifier]*ast.LoadedPackage,
	out string, debug, upgrade bool, cacheDir string,
	log *common.LogWriter,
) error {
	return nil
}

func (l EmptyLinker) Cleanup() {
}

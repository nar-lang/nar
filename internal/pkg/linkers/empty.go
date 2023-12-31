package linkers

import (
	"oak-compiler/internal/pkg/ast"
	"oak-compiler/internal/pkg/common"
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

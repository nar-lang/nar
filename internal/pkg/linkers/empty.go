package linkers

import (
	"io"
	"oak-compiler/internal/pkg/ast"
)

type EmptyLinker struct {
}

func (l EmptyLinker) GetOutFileLocation(givenLocation string) string {
	return givenLocation + ".acorn"
}

func (l EmptyLinker) Link(main string, packages []ast.LoadedPackage, out string, debug, upgrade bool, cacheDir string, log io.Writer) error {
	return nil
}

func (l EmptyLinker) Cleanup() {
}

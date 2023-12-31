package oakc

import (
	"oak-compiler/internal/pkg/ast"
	"oak-compiler/internal/pkg/common"
	"oak-compiler/internal/pkg/linkers"
)

type Linker interface {
	GetOutFileLocation(givenLocation string) string
	Link(
		main ast.FullIdentifier, packages map[ast.PackageIdentifier]*ast.LoadedPackage,
		out string, debug, upgrade bool, cacheDir string,
		log *common.LogWriter,
	) error
	Cleanup()
}

func GetLinker(name string) Linker {
	switch name {
	case "js":
		return &linkers.JsLinker{}
	}
	return linkers.EmptyLinker{}

}

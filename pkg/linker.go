package oakc

import (
	"io"
	"oak-compiler/internal/pkg/ast"
	"oak-compiler/internal/pkg/linkers"
)

type Linker interface {
	GetOutFileLocation(givenLocation string) string
	Link(main string, packages []ast.LoadedPackage, out string, debug, upgrade bool, cacheDir string, log io.Writer) error
	Cleanup()
}

func GetLinker(name string) Linker {
	switch name {
	case "js":
		return &linkers.JsLinker{}
	}
	return linkers.EmptyLinker{}

}

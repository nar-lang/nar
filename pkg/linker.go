package narc

import (
	"log"
	"nar-compiler/pkg/bytecode"
	"nar-compiler/pkg/locator"
)

type Linker interface {
	Link(log *log.Logger, binary *bytecode.Binary, lc locator.Locator, debug bool, outPath string) error
}

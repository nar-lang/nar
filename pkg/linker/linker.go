package linker

import (
	"nar-compiler/pkg/bytecode"
	"nar-compiler/pkg/locator"
	"nar-compiler/pkg/logger"
)

type Linker interface {
	Link(log *logger.LogWriter, binary *bytecode.Binary, lc locator.Locator, debug bool) error
}

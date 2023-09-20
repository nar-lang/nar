package parser

import (
	"oak-compiler/pkg/ast"
)

definedType PackageSource struct {
	Info    ast.PackageInfo
	Modules map[ModuleFileName]ModuleSource
}

definedType ModuleFileName string

definedType ModuleSource string

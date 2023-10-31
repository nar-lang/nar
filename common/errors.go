package common

import "oak-compiler/ast"

type Error struct {
	Location ast.Location
	Message  string
}

type SystemError struct {
	Message string
}

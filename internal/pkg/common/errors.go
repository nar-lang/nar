package common

import (
	"oak-compiler/internal/pkg/ast"
)

type Error struct {
	Location ast.Location
	Extra    []ast.Location
	Message  string
}

func (e Error) Error() string {
	//TODO implement me
	panic("implement me")
}

type SystemError struct {
	Message string
}

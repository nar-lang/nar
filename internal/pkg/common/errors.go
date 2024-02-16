package common

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"runtime"
)

type WithLocation interface {
	Location() ast.Location
}

type ErrorWithLocation interface {
	error
	WithLocation
	Message() string
}

func NewErrorOf(e WithLocation, msg string, params ...any) error {
	return NewErrorAt(e.Location(), msg, params...)
}

func NewErrorAt(loc ast.Location, msg string, params ...any) error {
	return locatedError{location: loc, message: fmt.Sprintf(msg, params...)}
}

type locatedError struct {
	location ast.Location
	message  string
}

func (e locatedError) Location() ast.Location {
	return e.location
}

func (e locatedError) Message() string {
	return e.message
}

func (e locatedError) Error() string {
	cursorString := e.location.CursorString()
	if cursorString != "" {
		return cursorString + " " + e.message
	}
	return e.message
}

func NewSystemError(err error) error {
	return systemError{inner: err}
}

type systemError struct {
	inner error
}

func (e systemError) Error() string {
	return fmt.Sprintf("system error: %s", e.inner.Error())
}

func NewCompilerError(message string) error {
	_, file, line, _ := runtime.Caller(1)
	return compilerError{message: message, file: file, line: line}
}

type compilerError struct {
	message string
	file    string
	line    int
}

func (e compilerError) Error() string {
	return fmt.Sprintf("%s at %s:%d", e.message, e.file, e.line)
}

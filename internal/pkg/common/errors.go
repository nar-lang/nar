package common

import (
	"fmt"
	"oak-compiler/internal/pkg/ast"
	"runtime"
	"slices"
	"strings"
)

type Error struct {
	Location ast.Location
	Extra    []ast.Location
	Message  string
}

func (e Error) Error() string {
	sb := strings.Builder{}
	if e.Location.FilePath != "" {
		line, col := e.Location.GetLineAndColumn()
		sb.WriteString(fmt.Sprintf("%s:%d:%d %s\n", e.Location.FilePath, line, col, e.Message))
	}

	var uniqueExtra []ast.Location
	for _, e := range e.Extra {
		if !slices.ContainsFunc(uniqueExtra, func(x ast.Location) bool {
			return x.FilePath == e.FilePath && x.Position == e.Position
		}) {
			uniqueExtra = append(uniqueExtra, e)
		}
	}

	for _, extra := range uniqueExtra {
		line, col := extra.GetLineAndColumn()
		sb.WriteString(fmt.Sprintf("  ->%s:%d:%d\n", extra.FilePath, line, col))
	}

	if e.Location.FilePath == "" {
		sb.WriteString(fmt.Sprintf("%s\n", e.Message))
	}
	return sb.String()
}

func NewSystemError(err error) error {
	return systemError{inner: err}
}

type systemError struct {
	inner error
}

func (e systemError) Error() string {
	return fmt.Sprintf("system error: %v", e.inner)
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

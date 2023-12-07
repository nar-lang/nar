package common

import (
	"fmt"
	"oak-compiler/internal/pkg/ast"
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
		sb.WriteString(fmt.Sprintf("  -> %s:%d:%d\n", extra.FilePath, line, col))
	}

	if e.Location.FilePath == "" {
		sb.WriteString(fmt.Sprintf("%s\n", e.Message))
	}
	return sb.String()
}

type SystemError struct {
	Message string
}

func (e SystemError) Error() string {
	return e.Message
}

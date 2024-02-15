package lsp

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/lsp/protocol"
	"strings"
)

func uriToPath(path protocol.DocumentURI) string {
	if strings.HasPrefix(string(path), "file://") {
		path = path[7:]
	}
	return string(path)
}

func pathToUri(path string) protocol.DocumentURI {
	return protocol.DocumentURI("file://" + path)
}

func locToRange(loc ast.Location) protocol.Range {
	line, c, eline, ec := loc.GetLineAndColumn()
	return protocol.Range{
		Start: protocol.Position{Line: uint32(line - 1), Character: uint32(c - 1)},
		End:   protocol.Position{Line: uint32(eline - 1), Character: uint32(ec - 1)},
	}
}

func locToLocation(loc ast.Location) *protocol.Location {
	return &protocol.Location{
		URI:   pathToUri(loc.FilePath()),
		Range: locToRange(loc),
	}
}

type withLocation interface {
	Location() ast.Location
}

type atLocation struct {
	loc  ast.Location
	stmt withLocation
}

func find[T withLocation](stmt T, x atLocation) atLocation {
	return findAtLocation(stmt, x)
}

func findAtLocation(stmt withLocation, x atLocation) atLocation {
	if stmt != nil && stmt.Location().Contains(x.loc) {
		if x.stmt == nil || x.stmt.Location().Size() > stmt.Location().Size() {
			x.stmt = stmt
		}
	}
	return x
}

type successors struct {
	loc   ast.Location
	stmts []withLocation
}

func findSuccessors[T withLocation](stmt T, x successors) successors {
	return findSuccessorsAtLocation(stmt, x)
}

func findSuccessorsAtLocation(stmt withLocation, x successors) successors {
	if stmt != nil && stmt.Location().Start() == x.loc.Start() && stmt.Location().End() == x.loc.End() {
		x.stmts = append(x.stmts, stmt)
	}
	return x
}

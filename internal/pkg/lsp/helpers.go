package lsp

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/ast/parsed"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/lsp/protocol"
)

func (s *server) locationUnderCursor(docURI protocol.DocumentURI, line, char uint32) (ast.Location, *parsed.Module, bool) {
	path := uriToPath(docURI)
	for _, m := range s.parsedModules {
		if m != nil && m.Location().FilePath() == path {
			loc := ast.NewLocationSrc(path, m.Location().FileContent(), line, char)
			return loc, m, true
		}
	}
	return ast.Location{}, nil, false
}

func (s *server) statementAtLocation(
	loc ast.Location, m *parsed.Module,
) (
	parsed.Statement, normalized.Statement, typed.Statement, *parsed.Module,
) {
	var pStmt parsed.Statement
	m.Iterate(func(x parsed.Statement) {
		if x != nil && x.Location().Contains(loc) && (pStmt == nil || pStmt.Location().Size() > x.Location().Size()) {
			pStmt = x
		}
	})
	if pStmt != nil {
		nStmt := pStmt.Successor()
		var tStmt typed.Statement
		if nStmt != nil {
			tStmt = nStmt.Successor()
		}
		return pStmt, nStmt, tStmt, m
	}
	return nil, nil, nil, nil
}

package lsp

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"pkg.nimblebun.works/go-lsp"
)

func (s *server) Initialize(params *lsp.InitializeParams) (lsp.InitializeResult, error) {
	s.rootURI = params.RootURI
	s.trace = params.Trace
	s.workspaceFolders = params.WorkspaceFolders

	return lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{
			TextDocumentSync: &lsp.TextDocumentSyncOptions{
				OpenClose: true,
				Change:    lsp.TDSyncKindFull,
			},
			HoverProvider: &lsp.HoverOptions{},
			DeclarationProvider: &lsp.DeclarationRegistrationOptions{
				DeclarationOptions: lsp.DeclarationOptions{},
				TextDocumentRegistrationOptions: lsp.TextDocumentRegistrationOptions{
					DocumentSelector: []lsp.DocumentFilter{
						{Pattern: "**/*.nar"},
					},
				},
				StaticRegistrationOptions: lsp.StaticRegistrationOptions{},
			},
		},
		ServerInfo: lsp.ServerInfo{
			Name:    "Nar Language Server",
			Version: "0.1.0",
		},
	}, nil
}

func (s *server) Initialized(_ *nothing) error {
	s.initialized = true
	return nil
}

func (s *server) Shutdown(_ *nothing) error {
	s.initialized = false
	return nil
}

func (s *server) S_setTraceNotification(params *traceNotificationParams) error {
	s.trace = params.Value
	return nil
}

func (s *server) TextDocument_didOpen(params *lsp.DidOpenTextDocumentParams) error {
	s.openedDocuments[params.TextDocument.URI] = &params.TextDocument
	s.compileChan <- docChange{params.TextDocument.URI, true}
	return nil
}

func (s *server) TextDocument_didChange(params *lsp.DidChangeTextDocumentParams) error {
	s.openedDocuments[params.TextDocument.URI].Text = params.ContentChanges[0].Text
	s.compileChan <- docChange{params.TextDocument.URI, false}
	return nil

}

func (s *server) TextDocument_didClose(params *lsp.DidCloseTextDocumentParams) error {
	delete(s.openedDocuments, params.TextDocument.URI)
	return nil
}

/*func (s *server) TextDocument_declaration(params *lsp.DeclarationParams) (*lsp.Location, error) {
	if doc, ok := s.openedDocuments[params.TextDocument.URI]; ok {
		loc := ast.NewLocationSrc(
			uriToPath(doc.URI),
			[]rune(doc.Text),
			params.Position.Line,
			params.Position.Character)
		for _, m := range s.typedModules {
			if m.Location.FilePath() == loc.FilePath() {
				d, e, t := findStatement(loc, m)
				if t != nil {
					return locToLocation(t.GetLocation()), nil
				}
				if e != nil {
					if g, ok := e.(*typed.Global); ok {
						if pm, ok := s.parsedModules[g.ModuleName]; ok {
							if d, ok := common.Find(func(d parsed.Definition) bool { return d.Name == g.DefinitionName }, pm.Definitions); ok {
								return locToLocation(d.Location), nil
							}
						}

					}

				}
				if d != nil {
					return locToLocation(d.Location), nil
				}
			}
		}
	}
	return nil, nil
}*/

func (s *server) TextDocument_hover(params *lsp.HoverParams) (*lsp.Hover, error) {
	if doc, ok := s.openedDocuments[params.TextDocument.URI]; ok {
		loc := ast.NewLocationSrc(
			uriToPath(doc.URI),
			[]rune(doc.Text),
			params.Position.Line,
			params.Position.Character)
		for _, m := range s.parsedModules {
			if m.Location.FilePath() == loc.FilePath() {
				d, e, t := findStatement(loc, m)
				if t != nil {
					/*return &lsp.Hover{
						Contents: lsp.MarkupContent{
							Kind:  lsp.MKPlainText,
							Value: t.String(),
						},
						Range: locToRange(t.GetLocation()),
					}, nil*/
				}
				if e != nil {
					return &lsp.Hover{
						Contents: lsp.MarkupContent{
							Kind:  lsp.MKPlainText,
							Value: getHelp(e),
						},
						Range: locToRange(e.GetLocation()),
					}, nil
					/*if g, ok := e.(*typed.Global); ok {
						if pm, ok := s.parsedModules[g.ModuleName]; ok {
							if d, ok := common.Find(func(d parsed.Definition) bool { return d.Name == g.DefinitionName }, pm.Definitions); ok {
								return &lsp.Hover{
									Contents: lsp.MarkupContent{
										Kind:  lsp.MKPlainText,
										Value: string(d.Name),
									},
									Range: locToRange(d.Location),
								}, nil
							}
						}

					}*/

				}
				if d != nil {
					return &lsp.Hover{
						Contents: lsp.MarkupContent{
							Kind:  lsp.MKPlainText,
							Value: fmt.Sprintf("definition of `%s`", d.Name),
						},
						Range: locToRange(d.Location),
					}, nil
				}
			}
		}
	}
	return nil, nil
}

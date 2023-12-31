package lsp

import (
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
		},
		ServerInfo: lsp.ServerInfo{
			Name:    "Oak Language Server",
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
	s.compileChan <- params.TextDocument.URI
	return nil
}

func (s *server) TextDocument_didChange(params *lsp.DidChangeTextDocumentParams) error {
	s.openedDocuments[params.TextDocument.URI].Text = params.ContentChanges[0].Text
	s.compileChan <- params.TextDocument.URI
	return nil

}

func (s *server) TextDocument_didClose(params *lsp.DidCloseTextDocumentParams) error {
	delete(s.openedDocuments, params.TextDocument.URI)
	return nil
}

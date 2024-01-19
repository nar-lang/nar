package lsp

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
	"nar-compiler/internal/pkg/lsp/protocol"
)

func (s *server) Initialize(params *protocol.InitializeParams) (protocol.InitializeResult, error) {
	s.rootURI = params.RootURI
	s.workspaceFolders = params.WorkspaceFolders
	if params.Trace != nil {
		s.trace = *params.Trace
	}

	return protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			TextDocumentSync: &protocol.TextDocumentSyncOptions{
				OpenClose: true,
				Change:    protocol.Full,
			},

			CompletionProvider: nil,
			HoverProvider: &protocol.Or_ServerCapabilities_hoverProvider{
				Value: true,
			},
			SignatureHelpProvider: nil,
			DefinitionProvider: &protocol.Or_ServerCapabilities_definitionProvider{
				Value: protocol.DefinitionOptions{},
			},
		},
		ServerInfo: &protocol.PServerInfoMsg_initialize{
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

func (s *server) S_setTraceNotification(params *protocol.SetTraceParams) error {
	s.trace = params.Value
	return nil
}

func (s *server) TextDocument_didOpen(params *protocol.DidOpenTextDocumentParams) error {
	s.openedDocuments[params.TextDocument.URI] = &params.TextDocument
	s.compileChan <- docChange{params.TextDocument.URI, true}
	return nil
}

func (s *server) TextDocument_didChange(params *protocol.DidChangeTextDocumentParams) error {
	s.openedDocuments[params.TextDocument.URI].Text = params.ContentChanges[0].Text
	s.compileChan <- docChange{params.TextDocument.URI, false}
	return nil

}

func (s *server) TextDocument_didClose(params *protocol.DidCloseTextDocumentParams) error {
	if pr, ok := s.documentToPackageRoot[params.TextDocument.URI]; ok {
		if pid, ok := s.packageRootToName[pr]; ok {
			if p, ok := s.loadedPackages[pid]; ok {
				for id, mod := range s.parsedModules {
					if mod.PackageName == p.Package.Name {
						delete(s.parsedModules, id)
						delete(s.normalizedModules, id)
						delete(s.typedModules, id)
					}
				}
				delete(s.loadedPackages, pid)
			}
			delete(s.packageRootToName, pr)
		}
		delete(s.documentToPackageRoot, params.TextDocument.URI)
	}
	delete(s.openedDocuments, params.TextDocument.URI)

	return nil
}

func (s *server) TextDocument_definition(params *protocol.DefinitionParams) (result *protocol.Location, err error) {
	_, wl := s.findDefinition(params.TextDocument.URI, params.Position.Line, params.Position.Character)
	if wl != nil {
		return locToLocation(wl.GetLocation()), nil
	}
	return nil, nil
}

func (s *server) TextDocument_hover(params *protocol.HoverParams) (*protocol.Hover, error) {
	var text string

	src, wl := s.findDefinition(params.TextDocument.URI, params.Position.Line, params.Position.Character)
	var moduleName ast.QualifiedIdentifier
	if wl != nil {
		for _, m := range s.parsedModules {
			if m.Location.FilePath() == wl.GetLocation().FilePath() {
				moduleName = m.Name
				break
			}
		}
	}
	switch wl.(type) {
	case *typed.Definition:
		{
			td := wl.(*typed.Definition)
			text = fmt.Sprintf("defined in `%s`\n\n```nar\ndef %s", moduleName, td.Name)
			um := typed.UnboundMap{}
			if len(td.Params) > 0 {
				text += "(" + common.Fold(func(p typed.Pattern, s string) string {
					if s != "" {
						s += ", "
					}
					return s + p.ToString(um, true, moduleName)
				}, "", td.Params) + "): "
				text += td.Expression.GetType().ToString(um, moduleName)
			} else if td.DeclaredType != nil {
				text += ": " + td.DeclaredType.ToString(um, moduleName)
			} else {
				text += ": " + td.GetType().ToString(um, moduleName)
			}
			text += "\n```"
		}
	case typed.Pattern:
		pt := wl.(typed.Pattern)

		um := typed.UnboundMap{}

		for _, tm := range s.typedModules {
			if tm.Location.FilePath() == pt.GetLocation().FilePath() {
				for _, d := range tm.Definitions {
					if d.Location.Contains(pt.GetLocation()) {
						d.GetType().ToString(um, moduleName)
					}
				}
			}
		}

		if do, ok := pt.(*typed.PDataOption); ok {
			text += fmt.Sprintf("```nar\n%s", do.OptionName)
			args := common.Fold(func(p typed.Pattern, s string) string {
				if s != "" {
					s += ", "
				}
				return s + p.ToString(um, true, moduleName)
			}, "", do.Args)
			if args != "" {
				text += "(" + args + ")"
			}
			text += ": " + pt.GetType().ToString(um, moduleName)
			text += "\n```"

		} else {
			text = fmt.Sprintf(
				"local variable\n```nar\n%s: %s\n```",
				src.GetLocation().Text(),
				pt.GetType().ToString(um, moduleName))
		}
	case typed.Type:
		t := wl.(typed.Type)
		text = fmt.Sprintf(
			"defined in `%s`\n\n```nar\n%s\n```", moduleName,
			t.ToString(typed.UnboundMap{}, moduleName))
		break

	}
	if text != "" {
		return &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: text,
			},
			Range: locToRange(src.GetLocation()),
		}, nil
	}
	return nil, nil
}

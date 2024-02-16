package lsp

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/parsed"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
	"nar-compiler/internal/pkg/lsp/protocol"
	"unicode"
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
			DefinitionProvider: &protocol.Or_ServerCapabilities_definitionProvider{
				Value: protocol.DefinitionOptions{},
			},
			TypeDefinitionProvider: &protocol.Or_ServerCapabilities_typeDefinitionProvider{
				Value: protocol.TypeDefinitionOptions{},
			},
			ReferencesProvider: &protocol.Or_ServerCapabilities_referencesProvider{
				Value: protocol.ReferenceOptions{},
			},
			HoverProvider: &protocol.Or_ServerCapabilities_hoverProvider{
				Value: true,
			},
			DocumentSymbolProvider: &protocol.Or_ServerCapabilities_documentSymbolProvider{
				Value: protocol.DocumentSymbolOptions{},
			},
			SemanticTokensProvider: &protocol.SemanticTokensOptions{
				Legend: protocol.SemanticTokensLegend{
					TokenTypes:     ast.SemanticTokenTypesLegend,
					TokenModifiers: ast.SemanticTokenModifiersLegend,
				},
				Range: &protocol.Or_SemanticTokensOptions_range{
					Value: true,
				},
				Full: &protocol.Or_SemanticTokensOptions_full{
					Value: protocol.PFullESemanticTokensOptions{
						Delta: false,
					},
				},
			},
		},
		ServerInfo: &protocol.PServerInfoMsg_initialize{
			Name:    "Nar Language Server",
			Version: Version,
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
					if mod.PackageName() == p.Package.Name {
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

func (s *server) TextDocument_definition(
	params *protocol.DefinitionParams,
) (result *protocol.Location, err error) {
	_, _, stmt, _ := s.statementAtLocation(params.TextDocument.URI, params.Position.Line, params.Position.Character)
	switch stmt.(type) {
	case *typed.Global:
		def := stmt.(*typed.Global).Definition()
		if def != nil {
			return locToLocation(def.Location()), nil
		}
	case *typed.Local:
		target := stmt.(*typed.Local).Target()
		if target != nil {
			return locToLocation(target.Location()), nil
		}
	case typed.Type:
		return locToLocation(stmt.Location()), nil
	case *typed.POption:
		def := stmt.(*typed.POption).Definition()
		if def != nil {
			return locToLocation(def.Location()), nil
		}
	}
	return nil, nil
}

func (s *server) TextDocument_typeDefinition(
	params *protocol.TypeDefinitionParams,
) (result *protocol.Location, err error) {
	_, _, stmt, _ := s.statementAtLocation(params.TextDocument.URI, params.Position.Line, params.Position.Character)
	switch stmt.(type) {
	case *typed.Global:
		t := stmt.(*typed.Global).Type()
		if t != nil {
			return locToLocation(t.Location()), nil
		}
	case *typed.Local:
		t := stmt.(*typed.Local).Type()
		if t != nil && t.Location().FilePath() != "" {
			return locToLocation(t.Location()), nil
		}
	case typed.Pattern:
		t := stmt.(typed.Pattern).Type()
		if t != nil {
			return locToLocation(t.Location()), nil
		}
	}
	return
}

func (s *server) TextDocument_references(
	params *protocol.ReferenceParams,
) (result []protocol.Location, err error) {
	_, _, stmt, _ := s.statementAtLocation(params.TextDocument.URI, params.Position.Line, params.Position.Character)

	appendDefinition := func(def *typed.Definition) {
		result = append(result, *locToLocation(def.Location()))
		for _, m := range s.parsedModules {
			m.Iterate(func(e parsed.Statement) {
				nStmt := e.Successor()
				if nStmt != nil {
					if g, ok := nStmt.Successor().(*typed.Global); ok {
						if g.Definition().Id() == def.Id() {
							result = append(result, *locToLocation(e.Location()))
						}
					}
				}
			})
		}
	}

	appendPattern := func(pattern typed.Pattern) {
		result = append(result, *locToLocation(pattern.Location()))
		for _, m := range s.parsedModules {
			if pattern.Location().FilePath() == m.Location().FilePath() {
				m.Iterate(func(stmt parsed.Statement) {
					nStmt := stmt.Successor()
					if nStmt != nil {
						tStmt := nStmt.Successor()
						if l, ok := tStmt.(*typed.Local); ok {
							if l.Target() == pattern {
								result = append(result, *locToLocation(stmt.Location()))
							}
						}
					}
				})
			}
		}
	}

	switch stmt.(type) {
	case *typed.Global:
		def := stmt.(*typed.Global).Definition()
		if def != nil {
			appendDefinition(def)
		}
		break
	case *typed.Local:
		target := stmt.(*typed.Local).Target()
		if target != nil {
			appendPattern(target)
		}
	case *typed.Definition:
		def := stmt.(*typed.Definition)
		appendDefinition(def)
		break
	case typed.Type:
		tNative, isNative := stmt.(*typed.TNative)
		tData, isData := stmt.(*typed.TData)
		if isNative || isData {
			for _, m := range s.parsedModules {
				m.Iterate(func(e parsed.Statement) {
					nStmt := e.Successor()
					if nStmt != nil {
						xNative, xIsNative := nStmt.Successor().(*typed.TNative)
						xData, xIsData := nStmt.Successor().(*typed.TData)
						if xIsNative && isNative && xNative.Name() == tNative.Name() {
							result = append(result, *locToLocation(e.Location()))
						}
						if xIsData && isData && xData.Name() == tData.Name() {
							result = append(result, *locToLocation(e.Location()))
						}
					}
				})
			}
		}
		break
	case typed.Pattern:
		pattern := stmt.(typed.Pattern)
		appendPattern(pattern)
		break
	}

	return
}

func (s *server) TextDocument_hover(params *protocol.HoverParams) (*protocol.Hover, error) {
	_, _, stmt, mod := s.statementAtLocation(params.TextDocument.URI, params.Position.Line, params.Position.Character)
	if stmt != nil {
		return &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: stmt.Code(mod.Name()), //TODO: make it useful
			},
			Range: locToRange(stmt.Location()),
		}, nil
	}
	return nil, nil
}

func (s *server) TextDocument_documentSymbol(
	params *protocol.DocumentSymbolParams,
) (result []protocol.DocumentSymbol, err error) {
	for _, mod := range s.parsedModules {
		if mod.Location().FilePath() == uriToPath(params.TextDocument.URI) {
			for _, inf := range mod.InfixFns() {
				result = append(result, protocol.DocumentSymbol{
					Name:           string(inf.Name()),
					Kind:           protocol.Operator,
					Range:          locToRange(inf.Location()),
					SelectionRange: locToRange(inf.Location()),
				})
			}
			for _, alias := range mod.Aliases() {
				findDT := func(x parsed.DataType) bool { return alias.Name() == x.Name() }
				if _, ok := common.Find(findDT, mod.DataTypes()); ok {
					continue
				}

				kind := protocol.Class
				var children []protocol.DocumentSymbol
				nType := alias.Successor()
				if nType != nil {
					tType := nType.Successor()
					switch tType.(type) {
					case nil:
						break
					case *typed.TRecord:
						kind = protocol.Struct
						for name, f := range tType.(*typed.TRecord).Fields() {
							children = append(children, protocol.DocumentSymbol{
								Name:           string(name),
								Kind:           protocol.Field,
								Range:          locToRange(f.Location()),
								SelectionRange: locToRange(f.Location()),
							})
						}

					case *typed.TFunc:
						kind = protocol.Function
					case *typed.TTuple:
						kind = protocol.Array
					case *typed.TNative:
						kind = protocol.Class
					case *typed.TUnbound:
						kind = protocol.Null

					}
					result = append(result, protocol.DocumentSymbol{
						Name:           string(alias.Name()),
						Kind:           kind,
						Range:          locToRange(alias.Location()),
						SelectionRange: locToRange(alias.Location()),
						Children:       children,
					})
				}
			}
			for _, dt := range mod.DataTypes() {
				result = append(result, protocol.DocumentSymbol{
					Name:           string(dt.Name()),
					Kind:           protocol.Enum,
					Range:          locToRange(dt.Location()),
					SelectionRange: locToRange(dt.Location()),
					Children: common.Map(func(o parsed.DataTypeOption) protocol.DocumentSymbol {
						return protocol.DocumentSymbol{
							Name:           string(o.Name()),
							Kind:           protocol.EnumMember,
							Range:          locToRange(o.Location()),
							SelectionRange: locToRange(o.Location()),
						}
					}, dt.Options()),
				})
			}
			for _, d := range mod.Definitions() {
				if unicode.IsLower([]rune(d.Name())[0]) {
					kind := protocol.Function
					if len(d.Params()) == 0 {
						kind = protocol.Constant
					}
					if _, ok := d.Body().(*parsed.Call); ok {
						kind = protocol.Interface
					}
					result = append(result, protocol.DocumentSymbol{
						Name:           string(d.Name()),
						Kind:           kind,
						Range:          locToRange(d.Location()),
						SelectionRange: locToRange(d.Location()),
					})
				}
			}
			break
		}
	}
	return
}

func (s *server) TextDocument_semanticTokens_range(
	params *protocol.SemanticTokensParams,
) (*protocol.SemanticTokensRangeParams, error) {
	s.reportError("Semantic tokens requested range")
	return nil, nil
}

func (s *server) TextDocument_semanticTokens_full(
	params *protocol.SemanticTokensParams,
) (*protocol.SemanticTokens, error) {
	s.reportError("Semantic tokens requested full")
	return nil, nil
}

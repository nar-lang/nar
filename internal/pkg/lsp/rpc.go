package lsp

import (
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
			HoverProvider: &protocol.Or_ServerCapabilities_hoverProvider{
				Value: true,
			},
			DefinitionProvider: &protocol.Or_ServerCapabilities_definitionProvider{
				Value: protocol.DefinitionOptions{},
			},
			DocumentSymbolProvider: &protocol.Or_ServerCapabilities_documentSymbolProvider{
				Value: protocol.DocumentSymbolOptions{},
			},
			ReferencesProvider: &protocol.Or_ServerCapabilities_referencesProvider{
				Value: protocol.ReferenceOptions{},
			},
			TypeDefinitionProvider: &protocol.Or_ServerCapabilities_typeDefinitionProvider{
				Value: protocol.TypeDefinitionOptions{},
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

//func (s *server) S_setTraceNotification(params *protocol.SetTraceParams) error {
//	s.trace = params.Value
//	return nil
//}
//
//func (s *server) TextDocument_didOpen(params *protocol.DidOpenTextDocumentParams) error {
//	s.openedDocuments[params.TextDocument.URI] = &params.TextDocument
//	s.compileChan <- docChange{params.TextDocument.URI, true}
//	return nil
//}
//
//func (s *server) TextDocument_didChange(params *protocol.DidChangeTextDocumentParams) error {
//	s.openedDocuments[params.TextDocument.URI].Text = params.ContentChanges[0].Text
//	s.compileChan <- docChange{params.TextDocument.URI, false}
//	return nil
//
//}
//
//func (s *server) TextDocument_didClose(params *protocol.DidCloseTextDocumentParams) error {
//	if pr, ok := s.documentToPackageRoot[params.TextDocument.URI]; ok {
//		if pid, ok := s.packageRootToName[pr]; ok {
//			if p, ok := s.loadedPackages[pid]; ok {
//				for id, mod := range s.parsedModules {
//					if mod.PackageName() == p.Package.name {
//						delete(s.parsedModules, id)
//						delete(s.normalizedModules, id)
//						delete(s.typedModules, id)
//					}
//				}
//				delete(s.loadedPackages, pid)
//			}
//			delete(s.packageRootToName, pr)
//		}
//		delete(s.documentToPackageRoot, params.TextDocument.URI)
//	}
//	delete(s.openedDocuments, params.TextDocument.URI)
//
//	return nil
//}
//
//func (s *server) TextDocument_definition(
//	params *protocol.DefinitionParams,
//) (result *protocol.location, err error) {
//	sl, wl := s.findDefinition(params.TextDocument.URI, params.Position.Line, params.Position.Character)
//	if wl != nil && sl != nil {
//		return locToLocation(wl.GetLocation()), nil
//	}
//	return nil, nil
//}
//
//func (s *server) TextDocument_typeDefinition(
//	params *protocol.TypeDefinitionParams,
//) (result *protocol.location, err error) {
//	_, wl := s.findDefinition(params.TextDocument.URI, params.Position.Line, params.Position.Character)
//	if _, ok := wl.(typed.Pattern); ok {
//		_, l := s.findStatementDefinition(wl, s.getModuleOfStatement(wl))
//		result = locToLocation(l.GetLocation())
//	}
//	return
//}
//
//func (s *server) TextDocument_hover(params *protocol.HoverParams) (*protocol.Hover, error) {
//	var text string
//
//	src, wl := s.findDefinition(params.TextDocument.URI, params.Position.Line, params.Position.Character)
//	var moduleName ast.QualifiedIdentifier
//	if wl != nil {
//		for _, m := range s.parsedModules {
//			if m.GetLocation().FilePath() == wl.GetLocation().FilePath() {
//				moduleName = m.name()
//				break
//			}
//		}
//	}
//	switch wl.(type) {
//	case *typed.Definition:
//		{
//			td := wl.(*typed.Definition)
//			text = fmt.Sprintf("defined in `%s`\n\n```nar\ndef %s", moduleName, td.name)
//			um := typed.UnboundMap{}
//			if len(td.Params) > 0 {
//				text += "(" + common.Fold(func(p typed.Pattern, s string) string {
//					if s != "" {
//						s += ", "
//					}
//					return s + p.ToString(um, true, moduleName)
//				}, "", td.Params) + "): "
//				text += td.Expression.GetType().ToString(um, moduleName)
//			} else if td.DeclaredType != nil {
//				text += ": " + td.DeclaredType.ToString(um, moduleName)
//			} else {
//				text += ": " + td.GetType().ToString(um, moduleName)
//			}
//			text += "\n```"
//		}
//	case typed.Pattern:
//		pt := wl.(typed.Pattern)
//
//		um := typed.UnboundMap{}
//
//		for _, tm := range s.typedModules {
//			if tm.location.FilePath() == pt.GetLocation().FilePath() {
//				for _, d := range tm.Definitions {
//					if d.location.Contains(pt.GetLocation()) {
//						d.GetType().ToString(um, moduleName)
//					}
//				}
//			}
//		}
//
//		if do, ok := pt.(*typed.PDataOption); ok {
//			text += fmt.Sprintf("```nar\n%s", do.OptionName)
//			args := common.Fold(func(p typed.Pattern, s string) string {
//				if s != "" {
//					s += ", "
//				}
//				return s + p.ToString(um, true, moduleName)
//			}, "", do.Args)
//			if args != "" {
//				text += "(" + args + ")"
//			}
//			text += ": " + pt.GetType().ToString(um, moduleName)
//			text += "\n```"
//
//		} else {
//			text = fmt.Sprintf(
//				"local variable\n```nar\n%s: %s\n```",
//				src.GetLocation().Text(),
//				pt.GetType().ToString(um, moduleName))
//		}
//	case typed.Type:
//		t := wl.(typed.Type)
//		text = fmt.Sprintf(
//			"defined in `%s`\n\n```nar\n%s\n```", moduleName,
//			t.ToString(typed.UnboundMap{}, moduleName))
//		break
//
//	}
//	if text != "" {
//		return &protocol.Hover{
//			Contents: protocol.MarkupContent{
//				Kind:  protocol.Markdown,
//				Value: text,
//			},
//			Range: locToRange(src.GetLocation()),
//		}, nil
//	}
//	return nil, nil
//}
//
//func (s *server) TextDocument_documentSymbol(
//	params *protocol.DocumentSymbolParams,
//) (result []protocol.DocumentSymbol, err error) {
//	//for _, mod := range s.parsedModules {
//	//	if mod.location.FilePath() == uriToPath(params.TextDocument.URI) {
//	//		for _, inf := range mod.InfixFns {
//	//			result = append(result, protocol.DocumentSymbol{
//	//				name:           string(inf.name),
//	//				Kind:           protocol.Operator,
//	//				Range:          locToRange(inf.location),
//	//				SelectionRange: locToRange(inf.location),
//	//			})
//	//		}
//	//		for _, alias := range mod.Aliases {
//	//			findDT := func(x *parsed.DataType) bool { return alias.name == x.name }
//	//			if _, ok := common.Find(findDT, mod.DataTypes); ok {
//	//				continue
//	//			}
//	//
//	//			kind := protocol.Class
//	//			var children []protocol.DocumentSymbol
//	//
//	//			if rec, ok := alias.type_.(*parsed.TRecord); ok {
//	//				kind = protocol.Struct
//	//				for name, f := range rec.fields() {
//	//					children = append(children, protocol.DocumentSymbol{
//	//						name:           string(name),
//	//						Kind:           protocol.Field,
//	//						Range:          locToRange(f.GetLocation()),
//	//						SelectionRange: locToRange(f.GetLocation()),
//	//					})
//	//				}
//	//			} else {
//	//				succ := typed.FoldModule(
//	//					findSuccessors[typed.Expression],
//	//					findSuccessors[typed.Type],
//	//					findSuccessors[typed.Pattern],
//	//					successors{loc: alias.location},
//	//					s.typedModules[mod.name])
//	//
//	//				for _, s := range succ.stmts {
//	//					if t, ok := s.(typed.Type); ok {
//	//						switch t.(type) {
//	//						case *typed.TFunc:
//	//							kind = protocol.Function
//	//						case *typed.TTuple:
//	//							kind = protocol.Array
//	//						case *typed.TNative:
//	//							kind = protocol.Class
//	//						case *typed.TUnbound:
//	//							kind = protocol.Null
//	//						}
//	//					}
//	//				}
//	//			}
//	//
//	//			result = append(result, protocol.DocumentSymbol{
//	//				name:           string(alias.name),
//	//				Kind:           kind,
//	//				Range:          locToRange(alias.location),
//	//				SelectionRange: locToRange(alias.location),
//	//				Children:       children,
//	//			})
//	//		}
//	//		for _, dt := range mod.DataTypes {
//	//			result = append(result, protocol.DocumentSymbol{
//	//				name:           string(dt.name),
//	//				Kind:           protocol.Enum,
//	//				Range:          locToRange(dt.location),
//	//				SelectionRange: locToRange(dt.location),
//	//				Children: common.Map(func(o parsed.DataTypeOption) protocol.DocumentSymbol {
//	//					return protocol.DocumentSymbol{
//	//						name:           string(o.name),
//	//						Kind:           protocol.EnumMember,
//	//						Range:          locToRange(o.location),
//	//						SelectionRange: locToRange(o.location),
//	//					}
//	//				}, dt.Options),
//	//			})
//	//		}
//	//		for _, d := range mod.Definitions {
//	//			if unicode.IsLower([]rune(d.name)[0]) {
//	//				kind := protocol.Function
//	//				if len(d.Params) == 0 {
//	//					kind = protocol.Constant
//	//				}
//	//				if _, ok := d.Expression.(*parsed.NativeCall); ok {
//	//					kind = protocol.Interface
//	//				}
//	//				result = append(result, protocol.DocumentSymbol{
//	//					name:           string(d.name),
//	//					Kind:           kind,
//	//					Range:          locToRange(d.location),
//	//					SelectionRange: locToRange(d.location),
//	//				})
//	//			}
//	//		}
//	//		break
//	//	}
//	//}
//	return
//}
//
//func (s *server) TextDocument_references(
//	params *protocol.ReferenceParams,
//) (result []protocol.location, err error) {
//	src, found := s.findDefinition(params.TextDocument.URI, params.Position.Line, params.Position.Character)
//	switch found.(type) {
//	case *typed.Definition:
//		def := found.(*typed.Definition)
//		result = append(result, *locToLocation(def.GetLocation()))
//		var moduleName ast.QualifiedIdentifier
//		for _, m := range s.parsedModules {
//			if m.GetLocation().FilePath() == def.GetLocation().FilePath() {
//				moduleName = m.name()
//				break
//			}
//		}
//		for _, m := range s.typedModules {
//			result = typed.FoldModule(
//				func(e typed.Expression, acc []protocol.location) []protocol.location {
//					if g, ok := e.(*typed.Global); ok {
//						if g.Definition.name == def.name && g.ModuleName == moduleName {
//							return append(acc, *locToLocation(e.GetLocation()))
//						}
//					}
//					return acc
//				},
//				func(t typed.Type, acc []protocol.location) []protocol.location { return acc },
//				func(p typed.Pattern, acc []protocol.location) []protocol.location { return acc },
//				result, m)
//		}
//		break
//	case typed.Type:
//		tNative, isNative := found.(*typed.TNative)
//		tData, isData := found.(*typed.TData)
//		if isNative || isData {
//			for _, m := range s.typedModules {
//				result = typed.FoldModule(
//					func(e typed.Expression, acc []protocol.location) []protocol.location {
//						return acc
//					},
//					func(t typed.Type, acc []protocol.location) []protocol.location {
//						xNative, xIsNative := t.(*typed.TNative)
//						xData, xIsData := t.(*typed.TData)
//						if xIsNative && isNative && xNative.name == tNative.name {
//							return append(acc, *locToLocation(t.GetLocation()))
//						}
//						if xIsData && isData && xData.name == tData.name {
//							return append(acc, *locToLocation(t.GetLocation()))
//						}
//						return acc
//					},
//					func(p typed.Pattern, acc []protocol.location) []protocol.location {
//						return acc
//					},
//					result, m)
//			}
//		}
//
//		break
//	case typed.Pattern:
//		loc := ast.NewLocationSrc(
//			src.GetLocation().FilePath(),
//			src.GetLocation().FileContent(),
//			params.Position.Line,
//			params.Position.Character,
//		)
//
//		//TODO: it does not take scope into account
//
//		pt := found.(typed.Pattern)
//		var name ast.Identifier
//		switch pt.(type) {
//		case *typed.PNamed:
//			name = pt.(*typed.PNamed).name
//			break
//		case *typed.PAlias:
//			name = pt.(*typed.PAlias).Alias
//			break
//		case *typed.PRecord:
//			rec := pt.(*typed.PRecord)
//			for _, f := range rec.fields {
//				if f.location.Contains(loc) {
//					name = f.name
//					break
//				}
//			}
//			break
//		}
//		if name != "" {
//			for _, m := range s.typedModules {
//				if m.location.FilePath() == src.GetLocation().FilePath() {
//					for _, d := range m.Definitions {
//						if d.location.Contains(loc) {
//							result = typed.FoldDefinition(
//								func(e typed.Expression, acc []protocol.location) []protocol.location {
//									if loc, ok := e.(*typed.Local); ok {
//										if loc.name == name {
//											return append(acc, *locToLocation(e.GetLocation()))
//										}
//									}
//									return acc
//								},
//								func(e typed.Type, acc []protocol.location) []protocol.location {
//									return acc
//								},
//								func(e typed.Pattern, acc []protocol.location) []protocol.location {
//									if n, ok := e.(*typed.PNamed); ok {
//										if n.name == name {
//											return append(acc, *locToLocation(e.GetLocation()))
//										}
//									}
//									if n, ok := e.(*typed.PAlias); ok {
//										if n.Alias == name {
//											return append(acc, *locToLocation(e.GetLocation()))
//										}
//									}
//									return acc
//								},
//								result, d)
//						}
//					}
//				}
//			}
//		}
//		break
//	}
//
//	return
//}

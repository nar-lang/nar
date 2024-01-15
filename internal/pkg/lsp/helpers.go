package lsp

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/ast/parsed"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
	"nar-compiler/internal/pkg/lsp/protocol"
)

func (s *server) findDefinition(docURI protocol.DocumentURI, line, char uint32) (source, result withLocation) {
	if doc, ok := s.openedDocuments[docURI]; ok {
		loc := ast.NewLocationSrc(uriToPath(doc.URI), []rune(doc.Text), line, char)
		for _, m := range s.parsedModules {
			if m.Location.FilePath() == loc.FilePath() {
				at := atLocation{loc: loc}
				at = parsed.FoldModule(find[parsed.Expression], find[parsed.Type], find[parsed.Pattern], at, m)

				switch at.stmt.(type) {
				case parsed.Expression:
					e := at.stmt.(parsed.Expression)
					if succ := e.GetSuccessor(); succ != nil {
						switch succ.(type) {
						case normalized.Global:
							{
								g := succ.(normalized.Global)
								if tm, ok := s.typedModules[g.ModuleName]; ok {
									td, ok := common.Find(
										func(d *typed.Definition) bool {
											return d.Name == g.DefinitionName
										},
										tm.Definitions)
									if ok {
										source = at.stmt
										result = td
									}
								}
							}
						case normalized.Local:
							{
								l := succ.(normalized.Local)
								if tm, ok := s.typedModules[m.Name]; ok && l.Target != nil {
									at := atLocation{loc: l.Target.GetLocation()}
									at = typed.FoldModule(
										find[typed.Expression],
										find[typed.Type],
										find[typed.Pattern],
										at, tm)

									if tp, ok := at.stmt.(typed.Pattern); ok {
										source = l
										result = tp
									}
								}
							}
						}
					}
					break
				case parsed.Pattern:
					p := at.stmt.(parsed.Pattern)
					succ := typed.FoldModule(
						findSuccessors[typed.Expression],
						findSuccessors[typed.Type],
						findSuccessors[typed.Pattern],
						successors{loc: p.GetLocation()},
						s.typedModules[m.Name])
					for _, stmt := range succ.stmts {
						if pt, ok := stmt.(typed.Pattern); ok {
							switch pt.(type) {
							case *typed.PDataOption:
								e := pt.(*typed.PDataOption)
								result = e.Definition
								source = p
								break
							case *typed.PNamed:
								result = pt
								source = p
							case *typed.PAlias:
								result = pt
								source = p
							default:

							}
							break
						}
					}
					break
				}
			}
		}
	}
	return
}

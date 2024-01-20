package lsp

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/ast/parsed"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
	"nar-compiler/internal/pkg/lsp/protocol"
	"nar-compiler/internal/pkg/processors"
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
										return
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
										return
									}
								}
							}
						}
					}
					break
				case parsed.Type:
					t := at.stmt.(parsed.Type)
					switch t.(type) {
					case *parsed.TNamed:
						nt := t.(*parsed.TNamed)
						x, _, _, _ := processors.FindParsedType(s.parsedModules, m, nt.Name, nt.Args, nt.Location)
						if x != nil {
							succ := typed.FoldModule(
								findSuccessors[typed.Expression],
								findSuccessors[typed.Type],
								findSuccessors[typed.Pattern],
								successors{loc: x.GetLocation()},
								s.typedModules[m.Name])
							for _, stmt := range succ.stmts {
								switch stmt.(type) {
								case typed.Type:
									source = t
									result = stmt
									return
								}
							}
						}
					}
				case parsed.Pattern:
					succ := typed.FoldModule(
						findSuccessors[typed.Expression],
						findSuccessors[typed.Type],
						findSuccessors[typed.Pattern],
						successors{loc: at.stmt.GetLocation()},
						s.typedModules[m.Name])
					for _, stmt := range succ.stmts {
						switch stmt.(type) {
						case typed.Pattern:
							pt := stmt.(typed.Pattern)
							switch pt.(type) {
							case *typed.PDataOption:
								e := pt.(*typed.PDataOption)
								result = e.Definition
								source = at.stmt
								return
							case *typed.PNamed:
								result = pt
								source = at.stmt
								return
							case *typed.PAlias:
								result = pt
								source = at.stmt
								return
							}
							break
						}
					}
					break
				case nil:
					for _, pDef := range m.Definitions {
						if pDef.Location.Contains(loc) {
							for _, tDef := range s.typedModules[m.Name].Definitions {
								if tDef.Name == pDef.Name {
									result = tDef
									source = pDef
									return
								}
							}
						}
					}
					break
				}
				break
			}
		}
	}
	return
}

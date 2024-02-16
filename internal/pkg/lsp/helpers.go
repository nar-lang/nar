package lsp

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/ast/parsed"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/lsp/protocol"
)

func (s *server) statementAtLocation(
	docURI protocol.DocumentURI, line, char uint32,
) (
	parsed.Statement, normalized.Statement, typed.Statement, *parsed.Module,
) {
	path := uriToPath(docURI)
	for _, m := range s.parsedModules {
		if m != nil && m.Location().FilePath() == path {
			loc := ast.NewLocationSrc(path, m.Location().FileContent(), line, char)
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
		}
	}
	return nil, nil, nil, nil
}

//
//func (s *server) findDefinition(docURI protocol.DocumentURI, line, char uint32) (source, result withLocation) {
//	stmt, m := s.statementAtLocation(docURI, line, char)
//	return s.findStatementDefinition(stmt, m)
//}
//
//func (s *server) findStatementDefinition(stmt withLocation, m *parsed.Module) (source, result withLocation) {
//	//switch stmt.(type) {
//	//case parsed.Expression:
//	//	e := stmt.(parsed.Expression)
//	//	if succ := e.Successor(); succ != nil {
//	//		switch succ.(type) {
//	//		case normalized.Global:
//	//			{
//	//				g := succ.(normalized.Global)
//	//				if tm, ok := s.typedModules[g.ModuleName]; ok {
//	//					td, ok := common.Find(
//	//						func(d *typed.Definition) bool {
//	//							return d.name == g.DefinitionName
//	//						},
//	//						tm.Definitions)
//	//					if ok {
//	//						source = stmt
//	//						result = td
//	//						return
//	//					}
//	//				}
//	//			}
//	//		case normalized.Local:
//	//			{
//	//				l := succ.(normalized.Local)
//	//				if tm, ok := s.typedModules[m.name()]; ok && l.Target != nil {
//	//					at := atLocation{loc: l.Target.Location()}
//	//					at = typed.FoldModule(
//	//						find[typed.Expression],
//	//						find[typed.Type],
//	//						find[typed.Pattern],
//	//						at, tm)
//	//
//	//					if tp, ok := at.stmt.(typed.Pattern); ok {
//	//						source = l
//	//						result = tp
//	//						return
//	//					}
//	//				}
//	//			}
//	//		}
//	//	}
//	//	break
//	//case parsed.Type:
//	//	t := stmt.(parsed.Type)
//	//	switch t.(type) {
//	//	case *parsed.TNamed:
//	//		nt := t.(*parsed.TNamed)
//	//		x, _, _, _ := nt.Find(s.parsedModules, m)
//	//		if x != nil {
//	//			succ := typed.FoldModule(
//	//				findSuccessors[typed.Expression],
//	//				findSuccessors[typed.Type],
//	//				findSuccessors[typed.Pattern],
//	//				successors{loc: x.Location()},
//	//				s.typedModules[m.name()])
//	//			for _, stmt := range succ.stmts {
//	//				switch stmt.(type) {
//	//				case typed.Type:
//	//					source = t
//	//					result = stmt
//	//					return
//	//				}
//	//			}
//	//		}
//	//	}
//	//case parsed.Pattern:
//	//	succ := typed.FoldModule(
//	//		findSuccessors[typed.Expression],
//	//		findSuccessors[typed.Type],
//	//		findSuccessors[typed.Pattern],
//	//		successors{loc: stmt.Location()},
//	//		s.typedModules[m.name()])
//	//	for _, stmt := range succ.stmts {
//	//		switch stmt.(type) {
//	//		case typed.Pattern:
//	//			pt := stmt.(typed.Pattern)
//	//			switch pt.(type) {
//	//			case *typed.PDataOption:
//	//				e := pt.(*typed.PDataOption)
//	//				result = e.Definition
//	//				source = stmt
//	//				return
//	//			case *typed.PNamed:
//	//				result = pt
//	//				source = stmt
//	//				return
//	//			case *typed.PAlias:
//	//				result = pt
//	//				source = stmt
//	//				return
//	//			}
//	//			break
//	//		}
//	//	}
//	//	break
//	//case nil:
//	//	//for _, pDef := range m.definitions {
//	//	//	if pDef.location.Contains(stmt.Location()) {
//	//	//		for _, tDef := range s.typedModules[m.name()].Definitions {
//	//	//			if tDef.name == pDef.name {
//	//	//				result = tDef
//	//	//				source = pDef
//	//	//				return
//	//	//			}
//	//	//		}
//	//	//	}
//	//	//}
//	//	break
//	//}
//	return
//}
//
//func (s *server) getModuleOfStatement(stmt withLocation) *parsed.Module {
//	for _, m := range s.parsedModules {
//		if m.Location().FilePath() == stmt.Location().FilePath() {
//			return m
//		}
//	}
//	return nil
//}

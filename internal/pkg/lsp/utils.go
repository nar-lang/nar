package lsp

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/parsed"
	"pkg.nimblebun.works/go-lsp"
	"strings"
)

func uriToPath(path lsp.DocumentURI) string {
	if strings.HasPrefix(string(path), "file://") {
		path = path[7:]
	}
	return string(path)
}

func pathToUri(path string) lsp.DocumentURI {
	return lsp.DocumentURI("file://" + path)
}

func locToRange(loc ast.Location) *lsp.Range {
	line, c, eline, ec := loc.GetLineAndColumn()
	return &lsp.Range{
		Start: lsp.Position{Line: line - 1, Character: c - 1},
		End:   lsp.Position{Line: eline - 1, Character: ec - 1},
	}
}

func locToLocation(loc ast.Location) *lsp.Location {
	return &lsp.Location{
		URI:   pathToUri(loc.FilePath()),
		Range: *locToRange(loc),
	}
}

func getHelp(expr parsed.Expression) string {
	switch expr.(type) {
	case parsed.Access:
		return "record field access"
	case parsed.Apply:
		return "function application"
	case parsed.Const:
		return "constant value"
	case parsed.If:
		return "if ... then ... else"
	case parsed.LetMatch:
		return "pattern match with local variable definition"
	case parsed.LetDef:
		return "local function definition"
	case parsed.List:
		return "constant list value"
	case parsed.Record:
		return "constant record value"
	case parsed.Select:
		return "select ... [case ... -> ...] end"
	case parsed.Tuple:
		return "constant tuple value"
	case parsed.Update:
		return "create a new record with updated fields"
	case parsed.Lambda:
		return "lambda definition"
	case parsed.Accessor:
		return "record field accessor"
	case parsed.BinOp:
		return "binary operator"
	case parsed.Negate:
		return "negate operator"
	case parsed.Constructor:
		return "type constructor"
	case parsed.InfixVar:
		return "infix function"
	case parsed.Var:
		return "identifier"
	case parsed.NativeCall:
		return "native function call"
	}
	return ""
}

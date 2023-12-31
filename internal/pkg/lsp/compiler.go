package lsp

import (
	"errors"
	"oak-compiler/internal/pkg/ast"
	"oak-compiler/internal/pkg/common"
	"oak-compiler/internal/pkg/processors"
	"os"
	"path/filepath"
	"pkg.nimblebun.works/go-lsp"
	"strings"
	"time"
)

func (s *server) compiler() {
	modifiedDocs := map[lsp.DocumentURI]struct{}{}
	modifiedPackages := map[ast.PackageIdentifier]struct{}{}
	for {
		waitTimeout := true
		for waitTimeout {
			select {
			case docUri := <-s.compileChan:
				modifiedDocs[docUri] = struct{}{}
				continue
			case <-time.After(500 * time.Millisecond):
				waitTimeout = false
				break
			}
		}

		if len(modifiedDocs) == 0 {
			continue
		}

		for name, mod := range s.parsedModules {
			for uri := range modifiedDocs {
				if "file://"+mod.Location.FilePath == string(uri) {
					delete(s.parsedModules, name)
					delete(s.normalizedModules, name)
					delete(s.typedModules, name)
					modifiedPackages[mod.PackageName] = struct{}{}
					delete(modifiedDocs, uri)
					break
				}
			}
		}

		for uri := range modifiedDocs {
			delete(modifiedDocs, uri)
			pkgName := s.getPackageOfDocument(uri)
			if pkgName != "" {
				modifiedPackages[pkgName] = struct{}{}
			}
		}

		s.compile(common.Keys(modifiedPackages))

		for name := range modifiedPackages {
			delete(modifiedPackages, name)
		}
	}
}

func findPackageRoot(path string) string {
	for path != "." && path != "/" {
		path = filepath.Dir(path)
		if _, err := os.Stat(filepath.Join(path, "oak.json")); !os.IsNotExist(err) {
			return path
		}
	}
	return ""
}

func (s *server) getPackageOfDocument(path lsp.DocumentURI) ast.PackageIdentifier {
	pkgRoot, ok := s.documentToPackageRoot[path]
	if !ok {
		safePath := string(path)
		if strings.HasPrefix(safePath, "file://") {
			safePath = safePath[7:]
		}
		pkgRoot = findPackageRoot(safePath)
		if pkgRoot != "" {
			s.documentToPackageRoot[path] = pkgRoot
		}
	}
	if pkgRoot == "" {
		return ""
	}

	pkgName, ok := s.packageRootToName[pkgRoot]
	if !ok {
		progress := func(_ float32, msg string) {
			s.notify("window/showMessage", lsp.ShowMessageParams{
				Type: lsp.MTInfo, Message: msg,
			})
		}
		pkg, err := processors.LoadPackage(pkgRoot, s.cacheDir, "", progress, false, s.loadedPackages)
		if err != nil {
			s.log.Err(err)
		}
		if pkg != nil {
			pkgName = pkg.Package.Name
			s.packageRootToName[pkgRoot] = pkgName
		}
	}

	return pkgName
}

func (s *server) compile(pkgNames []ast.PackageIdentifier) {
	log := &common.LogWriter{}

	affectedModuleNames := processors.Compile(
		pkgNames,
		s.loadedPackages,
		s.parsedModules,
		s.normalizedModules,
		s.typedModules,
		log,
		func(modulePath string) string {
			if doc, ok := s.openedDocuments[lsp.DocumentURI("file://"+modulePath)]; ok {
				return doc.Text
			}
			return ""
		})

	diagnosticData := s.extractDiagnosticsData(log)
	if len(diagnosticData) == 0 {
		s.log.Flush(os.Stdout)
	}

	for _, moduleName := range affectedModuleNames {
		if mod, ok := s.parsedModules[moduleName]; ok {
			uri := lsp.DocumentURI("file://" + mod.Location.FilePath)
			if _, reported := diagnosticData[uri]; !reported {
				s.notify("textDocument/publishDiagnostics", lsp.PublishDiagnosticsParams{
					URI:         uri,
					Diagnostics: []lsp.Diagnostic{},
				})
			}
		}
	}

	for uri, dsx := range diagnosticData {
		s.notify("textDocument/publishDiagnostics", lsp.PublishDiagnosticsParams{
			URI:         uri,
			Diagnostics: dsx,
		})
	}
}

func (s *server) extractDiagnosticsData(log *common.LogWriter) map[lsp.DocumentURI][]lsp.Diagnostic {
	diagnosticsData := map[lsp.DocumentURI][]lsp.Diagnostic{}

	insertDiagnostic := func(e common.Error, severity lsp.DiagnosticSeverity) {
		if e.Location.FilePath == "" {
			e.Location = e.Extra[0]
			e.Extra = e.Extra[1:]
		}
		uri := lsp.DocumentURI("file://" + e.Location.FilePath)
		line, c := e.Location.GetLineAndColumn()
		diagnosticsData[uri] = append(diagnosticsData[uri], lsp.Diagnostic{
			Range: lsp.Range{
				Start: lsp.Position{Line: line - 1, Character: c - 1},
				End:   lsp.Position{Line: line - 1, Character: c},
			},
			Severity: lsp.DSError,
			Message:  e.Message,
			RelatedInformation: common.Map(func(l ast.Location) lsp.DiagnosticRelatedInformation {
				line, c := e.Location.GetLineAndColumn()
				return lsp.DiagnosticRelatedInformation{
					Location: lsp.Location{
						URI: lsp.DocumentURI("file://" + l.FilePath),
						Range: lsp.Range{
							Start: lsp.Position{Line: line - 1, Character: c - 1},
							End:   lsp.Position{Line: line - 1, Character: c},
						},
					},
					Message: "?",
				}
			}, e.Extra),
		})
	}

	for _, err := range log.Errors() {
		var e common.Error
		if errors.As(err, &e) {
			insertDiagnostic(e, lsp.DSError)
		} else {
			s.log.Err(err)
		}
	}

	for _, err := range log.Warnings() {
		var e common.Error
		if errors.As(err, &e) {
			insertDiagnostic(e, lsp.DSWarning)
		} else {
			s.log.Warn(err)
		}
	}

	for _, msg := range log.Messages() {
		s.log.Info(msg)
	}

	return diagnosticsData
}

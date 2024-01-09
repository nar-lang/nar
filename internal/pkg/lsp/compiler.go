package lsp

import (
	"errors"
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
	"nar-compiler/internal/pkg/processors"
	"os"
	"path/filepath"
	"pkg.nimblebun.works/go-lsp"
	"time"
)

func (s *server) compiler() {
	modifiedDocs := map[lsp.DocumentURI]struct{}{}
	forcedDocs := map[lsp.DocumentURI]struct{}{}
	modifiedPackages := map[ast.PackageIdentifier]struct{}{}

	doc := <-s.compileChan
	modifiedDocs[doc.uri] = struct{}{}
	if doc.force {
		forcedDocs[doc.uri] = struct{}{}
	}

	for {
		waitTimeout := true
		for waitTimeout {
			select {
			case doc = <-s.compileChan:
				modifiedDocs[doc.uri] = struct{}{}
				if doc.force {
					forcedDocs[doc.uri] = struct{}{}
				}
				continue
			case <-time.After(500 * time.Millisecond):
				waitTimeout = false
				break
			}
		}

		if len(modifiedDocs) == 0 {
			continue
		}

		for uri := range forcedDocs {
			delete(forcedDocs, uri)
			s.getPackageOfDocument(uri, true)
		}

		for name, mod := range s.parsedModules {
			for uri := range modifiedDocs {
				path := uriToPath(uri)
				if mod.Location.FilePath() == path {
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
			pkgName := s.getPackageOfDocument(uri, false)
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
		if _, err := os.Stat(filepath.Join(path, "nar.json")); !os.IsNotExist(err) {
			return path
		}
	}
	return ""
}

func (s *server) getPackageOfDocument(uri lsp.DocumentURI, forceReload bool) ast.PackageIdentifier {
	pkgRoot, ok := s.documentToPackageRoot[uri]
	if !ok {
		path := uriToPath(uri)
		pkgRoot = findPackageRoot(path)
		if pkgRoot != "" {
			s.documentToPackageRoot[uri] = pkgRoot
		}
	}
	if pkgRoot == "" {
		return ""
	}

	pkgName, ok := s.packageRootToName[pkgRoot]
	if !ok || forceReload {
		if forceReload {
			delete(s.loadedPackages, pkgName)
		}
		progress := func(_ float32, msg string) {
			s.notify("window/showMessage", lsp.ShowMessageParams{
				Type: lsp.MTInfo, Message: msg,
			})
		}
		pkg, err := processors.LoadPackage(
			pkgRoot, s.cacheDir, "", progress, false, s.loadedPackages)
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
	defer func() {
		if r := recover(); r != nil {
			s.reportError(fmt.Sprintf("internal error:\n%v", r))
		}
	}()
	log := &common.LogWriter{}

	affectedModuleNames := processors.Compile(
		pkgNames,
		s.loadedPackages,
		s.parsedModules,
		s.normalizedModules,
		s.typedModules,
		log,
		func(modulePath string) string {
			if doc, ok := s.openedDocuments[pathToUri(modulePath)]; ok {
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
			uri := pathToUri(mod.Location.FilePath())
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
		if e.Location.IsEmpty() {
			e.Location = e.Extra[0]
			e.Extra = e.Extra[1:]
		}
		uri := pathToUri(e.Location.FilePath())
		diagnosticsData[uri] = append(diagnosticsData[uri], lsp.Diagnostic{
			Range:    *locToRange(e.Location),
			Severity: lsp.DSError,
			Message:  e.Message,
			RelatedInformation: common.Map(func(l ast.Location) lsp.DiagnosticRelatedInformation {
				return lsp.DiagnosticRelatedInformation{
					Location: *locToLocation(l),
					Message:  "?",
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

package linkers

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"io"
	"oak-compiler/internal/pkg/ast"
	"oak-compiler/internal/pkg/common"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const runtimeRepositoryUrl = "https://github.com/oaklang/runtime-js.git"
const nativeIndexPath = "native/js/index.js"

type JsLinker struct {
	artifactPath string
}

func (l *JsLinker) GetOutFileLocation(givenLocation string) string {
	return filepath.Join(givenLocation, "program.acorn")
}

func (l *JsLinker) Link(
	main string, packages []ast.LoadedPackage, out string, debug, upgrade bool, cacheDir string, log io.Writer,
) error {
	runtimePath, err := cacheRuntime(cacheDir, upgrade, log)
	if err != nil {
		return err
	}

	indexJs := strings.Builder{}
	var nativeNames []string
	indexJs.WriteString(fmt.Sprintf("import OakRuntime from '%s'\n", runtimePath))

	for _, pkg := range packages {
		_, err := os.Stat(path.Join(pkg.Dir, nativeIndexPath))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}

		jsName := dotToUnderscore(pkg.Package.Name)
		nativeNames = append(nativeNames, jsName)
		indexJs.WriteString(fmt.Sprintf("import %s from '%s'\n", jsName, filepath.Join(pkg.Dir, nativeIndexPath)))
	}

	indexJs.WriteString("const req = new XMLHttpRequest();\n")
	indexJs.WriteString("req.open('GET', 'program.acorn', true);\n")
	indexJs.WriteString("req.responseType = 'arraybuffer';\n")
	indexJs.WriteString("req.onload = function(e) {\n")
	indexJs.WriteString("    const runtime = new OakRuntime(e.target.response);\n")

	for _, name := range nativeNames {
		indexJs.WriteString(fmt.Sprintf("    %s(runtime);\n", name))
	}

	if main != "" {
		indexJs.WriteString(fmt.Sprintf("    runtime.execute('%s');\n", main))
	}

	indexJs.WriteString("};\nreq.send(null);\n")

	l.artifactPath = filepath.Join(out, "main.source.js")

	err = os.WriteFile(l.artifactPath, []byte(indexJs.String()), 0640)
	if err != nil {
		panic(common.SystemError{Message: err.Error()})
	}

	htmlPath := filepath.Join(out, "index.html")
	err = os.WriteFile(htmlPath, indexHtml, 0640)
	if err != nil {
		panic(common.SystemError{Message: err.Error()})
	}

	_, _ = fmt.Fprintf(log, "linked successfully\n")
	return nil
}

func (l *JsLinker) Cleanup() {
	_ = os.Remove(l.artifactPath)
}

func dotToUnderscore(s string) string {
	return strings.ReplaceAll(s, ".", "_")
}

func cacheRuntime(cacheDir string, upgrade bool, log io.Writer) (string, error) {
	runtimeDir, err := filepath.Abs(filepath.Join(cacheDir, "runtime-js"))
	if err != nil {
		return "", err
	}
	runtimePath := filepath.Join(runtimeDir, "index.js")
	_, err = os.Stat(runtimePath)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	loaded := err == nil

	if !loaded {
		_, _ = fmt.Fprintf(log, "cloning runtime `%s`\n", runtimeRepositoryUrl)
		_, err := git.PlainClone(runtimeDir, false, &git.CloneOptions{
			URL:      runtimeRepositoryUrl,
			Progress: log,
		})
		if err != nil {
			return "", err
		}
	} else if upgrade {
		r, err := git.PlainOpen(runtimeDir)
		if err == nil {
			_, _ = fmt.Fprintf(log, "upgrading runtime `%s` ", runtimeRepositoryUrl)
			w, err := r.Worktree()
			if err != nil {
				_, _ = fmt.Fprintf(log, "%s\n", err.Error())
			} else {
				err = w.Pull(&git.PullOptions{
					Progress: log,
				})
				if err != nil {
					_, _ = fmt.Fprintf(log, "%s\n", err.Error())
				} else {
					_, _ = fmt.Fprintf(log, "ok\n")
				}
			}
		}
	}
	return runtimePath, nil
}

var indexHtml = []byte(
	"<!DOCTYPE html>\n" +
		"<html lang=\"en\">\n" +
		"<head>\n" +
		"    <meta charset=\"UTF-8\">\n" +
		"    <title>Oak test</title>\n" +
		"    <script src=\"main.js\" type=\"module\"></script>\n" +
		"</head>\n" +
		"<body></body>\n" +
		"</html>\n")

package resolved

import (
	"errors"
	"fmt"
	cp "github.com/otiai10/copy"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func NewPackage(name PackageFullName, sourceDir string, modules map[string]Module, deps map[string]string) Package {
	return Package{
		name:      name,
		sourceDir: sourceDir,
		modules:   modules,
		deps:      deps,
	}
}

type Package struct {
	name      PackageFullName
	sourceDir string
	modules   map[string]Module
	deps      map[string]string
}

func (p Package) Write(outDir string) error {
	sb := &strings.Builder{}
	nativePath := path.Join(p.sourceDir, "native")
	outPath := path.Join(outDir, string(p.name))

	err := os.RemoveAll(outPath)
	if err != nil {
		return err
	}

	_, err = os.Stat(nativePath)
	if err == nil {
		err := cp.Copy(nativePath, outPath, cp.Options{
			Skip: func(info os.FileInfo, src, dest string) (bool, error) {
				_, fName := filepath.Split(src)
				return strings.HasPrefix(fName, "!"), nil
			},
		})
		if err != nil {
			return fmt.Errorf("failed to copy native directory `%s` to output path:\n%w", nativePath, err)
		}
	}

	err = os.MkdirAll(outPath, 0777)
	if err != nil {
		return fmt.Errorf("failed to make directory `%s`:\n%w", outPath, err)
	}

	for _, module := range p.modules {
		sb.Reset()
		module.write(sb)

		resultPath := path.Join(outPath, module.name+".gen.go")
		err = os.WriteFile(resultPath, []byte(sb.String()), 0666)
		if err != nil {
			return fmt.Errorf("failed to write moduleName file `%s`:\n%w", resultPath, err)
		}
	}

	gomodPath := path.Join(outDir, string(p.name), "go.mod")
	if _, err := os.Stat(gomodPath); errors.Is(err, os.ErrNotExist) {
		sb.Reset()
		sb.WriteString("module \"")
		sb.WriteString(string(p.name))
		sb.WriteString("\"\n\ngo 1.20\n\n")
		err = os.WriteFile(gomodPath, []byte(sb.String()), 0666)
		if err != nil {
			return fmt.Errorf("failed to write go.mod file:\n%w", err)
		}
	} else if err != nil {
		return err
	}

	sb.Reset()

	sb.WriteString("require \"github.com/oaklang/runtime\" ")
	sb.WriteString(kTargetRuntimeVersion)
	sb.WriteString("\n\n")

	for k := range p.deps {
		sb.WriteString("replace \"")
		sb.WriteString(k)
		sb.WriteString("\" => ")
		rel, _ := filepath.Rel(filepath.Dir(gomodPath), path.Join(outDir, k))
		sb.WriteString(rel)
		sb.WriteString("\nrequire \"")
		sb.WriteString(k)
		sb.WriteString("\" v0.0.0\n")
	}

	f, err := os.OpenFile(gomodPath, os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open go.mod file:\n%w", err)
	}
	_, err = f.Write([]byte(sb.String()))
	if err != nil {
		return fmt.Errorf("failed to write go.mod file:\n%w", err)
	}
	err = f.Close()
	if err != nil {
		return fmt.Errorf("failed to close go.mod file:\n%w", err)
	}

	return nil
}

func (p Package) Name() PackageFullName {
	return p.name
}

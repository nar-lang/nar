package lsp

import (
	"fmt"
	"maps"
	"nar-compiler/internal/pkg/lsp/protocol"
	"nar-compiler/pkg/locator"
	"path/filepath"
)

func newProvider(path string) *provider {
	return &provider{
		fsProvider: locator.NewFileSystemPackageProvider(path),
		overrides:  map[string][]rune{},
		merged:     map[string][]rune{},
		path:       path,
	}
}

type provider struct {
	fsProvider locator.Provider
	overrides  map[string][]rune
	merged     map[string][]rune
	path       string
	pkg        locator.Package
}

func (p *provider) ExportedPackages() ([]locator.Package, error) {
	if err := p.load(); err != nil {
		return nil, err
	}
	return []locator.Package{p.pkg}, nil
}

func (p *provider) LoadPackage(name string) (locator.Package, bool, error) {
	if err := p.load(); err != nil {
		return nil, false, err
	}
	if p.pkg.Info().Name == name {
		return p.pkg, true, nil
	}
	return nil, false, nil
}

func (p *provider) load() error {
	pkg, err := p.fsProvider.ExportedPackages()
	if err != nil {
		return err
	}
	if len(pkg) == 0 {
		return fmt.Errorf("failed to load package from %s", p.path)
	}

	for s := range p.merged {
		delete(p.merged, s)
	}

	maps.Copy(p.merged, pkg[0].Sources())
	maps.Copy(p.merged, p.overrides)
	p.pkg = locator.NewLoadedPackage(pkg[0].Info(), p.merged, filepath.Join(p.path, "native"))
	return nil
}

func (p *provider) OverrideFile(uri protocol.DocumentURI, content []rune) {
	path := uriToPath(uri)
	rel, _ := filepath.Rel(filepath.Join(p.path, "src"), path)
	if content == nil {
		delete(p.overrides, rel)
	} else {
		p.overrides[rel] = content
	}
}

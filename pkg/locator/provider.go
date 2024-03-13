package locator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Provider interface {
	ExportedPackages() ([]Package, error)
	LoadPackage(name string) (Package, bool, error)
}

func NewFileSystemPackageProvider(path string) Provider {
	return &fileSystemProvider{path: path}
}

type fileSystemProvider struct {
	path    string
	pkgInfo *PackageInfo
	pkgSrcs map[string][]rune
}

func (f *fileSystemProvider) ExportedPackages() ([]Package, error) {
	if err := f.loadInfo(); err != nil {
		return nil, err
	}
	if err := f.loadSources(); err != nil {
		return nil, err
	}
	return []Package{NewLoadedPackage(*f.pkgInfo, f.pkgSrcs, f.path)}, nil
}

func (f *fileSystemProvider) LoadPackage(name string) (Package, bool, error) {
	if err := f.loadInfo(); err != nil {
		return nil, false, err
	}
	if f.pkgInfo.Name == name {
		if err := f.loadSources(); err != nil {
			return nil, false, err
		}
		return NewLoadedPackage(*f.pkgInfo, f.pkgSrcs, f.path), true, nil
	}
	if strings.HasPrefix(name, ".") {
		provider := NewFileSystemPackageProvider(filepath.Join(f.path, name))
		packages, err := provider.ExportedPackages()
		if err != nil {
			return nil, false, err
		}
		if len(packages) > 0 {
			return packages[0], true, nil
		}
	}
	return nil, false, nil
}

func (f *fileSystemProvider) containsPackage() bool {
	_, err := os.Stat(filepath.Join(f.path, "nar.json"))
	return err == nil
}

func (f *fileSystemProvider) loadInfo() error {
	if f.pkgInfo == nil {
		infoFilePath := filepath.Join(f.path, "nar.json")
		infoBytes, err := os.ReadFile(infoFilePath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", infoFilePath, err)
		}
		var info PackageInfo
		if err = json.Unmarshal(infoBytes, &info); err != nil {
			return fmt.Errorf("failed to unmarshal %s: %w", infoFilePath, err)
		}
		f.pkgInfo = &info
	}
	return nil
}

func (f *fileSystemProvider) loadSources() error {
	if f.pkgSrcs == nil {
		f.pkgSrcs = map[string][]rune{}
		root := filepath.Join(f.path, "src")
		filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if filepath.Ext(path) != ".nar" {
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
			f.pkgSrcs[path] = []rune(string(content))
			return nil
		})
	}
	return nil
}

func NewMemoryPackageProvider(info PackageInfo, sources map[string][]rune) Provider {
	return &memoryProvider{
		pkg: NewLoadedPackage(info, sources, ""),
	}
}

type memoryProvider struct {
	pkg Package
}

func (m *memoryProvider) ExportedPackages() ([]Package, error) {
	return []Package{m.pkg}, nil
}

func (m *memoryProvider) LoadPackage(name string) (Package, bool, error) {
	if m.pkg.Info().Name == name {
		return m.pkg, true, nil
	}
	return nil, false, nil
}

func NewDirectoryProvider(root string) Provider {
	return &directoryProvider{root: root, pkgInfos: map[string]*fileSystemProvider{}}
}

type directoryProvider struct {
	root     string
	pkgInfos map[string]*fileSystemProvider
}

func (d *directoryProvider) ExportedPackages() ([]Package, error) {
	return nil, nil
}

func (d *directoryProvider) LoadPackage(name string) (Package, bool, error) {
	if err := d.load(); err != nil {
		return nil, false, err
	}
	if provider, ok := d.pkgInfos[name]; ok {
		return provider.LoadPackage(name)
	}
	return nil, false, nil
}

func (d *directoryProvider) load() error {
	dirs, err := os.ReadDir(d.root)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		provider := NewFileSystemPackageProvider(filepath.Join(d.root, dir.Name())).(*fileSystemProvider)
		if provider.containsPackage() {
			if err = provider.loadInfo(); err != nil {
				return err
			}
			d.pkgInfos[provider.pkgInfo.Name] = provider
		}
	}
	return nil
}

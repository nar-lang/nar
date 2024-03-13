package locator

import (
	"fmt"
	"nar-compiler/pkg/bytecode"
	"path/filepath"
	"slices"
	"strings"
)

func NewLocator(provider ...Provider) Locator {
	return &locator{providers: provider}
}

type Locator interface {
	Packages() ([]Package, error)
	FindPackage(name string) (Package, bool, error)
	EntryPoint() (bytecode.FullIdentifier, error)
}

type locator struct {
	providers []Provider
	packages  map[string]Package
}

func (l *locator) EntryPoint() (bytecode.FullIdentifier, error) {
	for _, provider := range l.providers {
		packages, err := provider.ExportedPackages()
		if err != nil {
			return "", err
		}
		for _, pkg := range packages {
			if pkg.Info().Main != "" {
				return bytecode.FullIdentifier(pkg.Info().Main), nil
			}
		}
	}
	return "", nil
}

func (l *locator) Packages() ([]Package, error) {
	if err := l.load(); err != nil {
		return nil, err
	}
	packages := make([]Package, 0, len(l.packages))
	for _, pkg := range l.packages {
		packages = append(packages, pkg)
	}
	slices.SortFunc(packages, func(a, b Package) int {
		if a.Info().Name < b.Info().Name {
			return -1
		} else {
			return 1
		}
	})

	return packages, nil
}

func (l *locator) load() error {
	l.packages = map[string]Package{}

	var addPackage func(pkg Package) error
	addPackage = func(pkg Package) error {
		if addedPackage, ok := l.packages[pkg.Info().Name]; ok {
			if addedPackage.Info().Version >= pkg.Info().Version {
				return nil
			}
		}
		l.packages[pkg.Info().Name] = pkg
		resolvedInfo := pkg.Info()
		resolvedInfo.Dependencies = map[string]int{}
		for depName, depVersion := range pkg.Info().Dependencies {
			var depPkg Package
			if depName == ".." || strings.HasPrefix(depName, "./") || strings.HasPrefix(depName, "../") {
				path := pkg.Path()
				if path != "" {
					exp, err := NewFileSystemPackageProvider(filepath.Join(path, depName)).ExportedPackages()
					if err != nil {
						return err
					}
					if len(exp) > 0 {
						depPkg = exp[0]
					}
				}
			} else {
				var ok bool
				var err error
				depPkg, ok, err = l.findDep(depName, depVersion)
				if err != nil {
					return err
				}
				if !ok {
					depPkg = nil
				}
			}

			if depPkg == nil {
				return fmt.Errorf(
					"package `%s` with version %d not found (dependency of %s)",
					depName, depVersion, pkg.Info().Name)
			}

			if err := addPackage(depPkg); err != nil {
				return err
			}
			resolvedInfo.Dependencies[depPkg.Info().Name] = depPkg.Info().Version
		}
		pkg.SetInfo(resolvedInfo)
		return nil
	}

	for _, provider := range l.providers {
		exported, err := provider.ExportedPackages()
		if err != nil {
			return err
		}
		for _, pkg := range exported {
			if err := addPackage(pkg); err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *locator) FindPackage(name string) (Package, bool, error) {
	if err := l.load(); err != nil {
		return nil, false, err
	}
	pkg, ok := l.packages[name]
	return pkg, ok, nil
}

func (l *locator) findDep(depName string, depVersion int) (Package, bool, error) {
	for _, provider := range l.providers {
		pkg, ok, err := provider.LoadPackage(depName)
		if err != nil {
			return nil, false, err
		}
		if ok {
			if depVersion <= pkg.Info().Version {
				return pkg, true, nil
			}
		}
	}
	return nil, false, nil
}

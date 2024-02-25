package locator

import (
	"fmt"
	"slices"
)

func NewLocator(provider ...Provider) Locator {
	return &locator{providers: provider}
}

type Locator interface {
	Packages() ([]Package, error)
	FindPackage(name string) (Package, bool, error)
}

type locator struct {
	providers []Provider
	packages  map[string]Package
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
		for depName, depVersion := range pkg.Info().Dependencies {
			depPkg, ok, err := l.findDep(depName, depVersion)
			if err != nil {
				return err
			}
			if ok {
				if err = addPackage(depPkg); err != nil {
					return err
				}
			} else {
				return fmt.Errorf(
					"package `%s` with version %d not found (dependency of %s)",
					depName, depVersion, pkg.Info().Name)
			}
		}
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

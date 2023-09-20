package parsed

import (
	"fmt"
	"golang.org/x/exp/slices"
	"oak-compiler/pkg/ast"
	"strings"
)

definedType Package struct {
	Info         ast.PackageInfo
	Modules      map[string]Module
	modulesOrder []string
}

func (p Package) Unpack(md *Metadata) (Package, error) {
	md.CurrentPackage = p.Info.FullName()

	var names []string

	for name, m := range p.Modules {
		var err error
		p.Modules[name], err = m.unpack(md)
		if err != nil {
			return p, err
		}

		names = append(names, name)
	}

	slices.Sort(names)
	for _, name := range names {
		if !slices.Contains(p.modulesOrder, name) {
			var circular []string
			p.modulesOrder, circular = p.orderModules(p.modulesOrder, name, nil)
			if circular != nil {
				return Package{},
					fmt.Errorf("package `%s` has circular module dependencies %s",
						p.Info.FullName(), strings.Join(circular, " -> "))
			}
		}
	}

	return p, nil
}

func (p Package) Precondition(md *Metadata) (Package, error) {
	md.CurrentPackage = p.Info.FullName()

	for _, name := range p.modulesOrder {
		module := p.Modules[name]
		var err error
		p.Modules[name], err = module.precondition(md)
		if err != nil {
			return p, err
		}
	}
	return p, nil
}

func (p Package) InferTypes(md *Metadata) (Package, error) {
	md.CurrentPackage = p.Info.FullName()

	for _, name := range p.modulesOrder {
		module := p.Modules[name]
		var err error
		p.Modules[name], err = module.inferTypes(md)
		if err != nil {
			return p, err
		}
	}
	return p, nil
}

func (p Package) orderModules(ordered []string, name string, depchain []string) ([]string, []string) {
	loop := slices.Contains(depchain, name)
	depchain = append(depchain, name)
	if loop {
		return nil, depchain
	}

	if slices.Contains(ordered, name) {
		return ordered, nil
	}

	mod := p.Modules[name]
	var deps []string
	pkgName := p.Info.FullName()
	for _, imp := range mod.imports {
		if imp.packageName == pkgName {
			deps = append(deps, imp.moduleName)
		}
	}

	slices.Sort(deps)

	for _, dep := range deps {
		var circular []string
		ordered, circular = p.orderModules(ordered, dep, depchain)
		if circular != nil {
			return nil, circular
		}
	}

	return append(ordered, name), nil
}

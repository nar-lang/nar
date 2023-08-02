package parsed

import "oak-compiler/pkg/resolved"

type Package struct {
	Dir     string
	Info    PackageInfo
	Modules map[string]Module
}

func (p Package) FullName() PackageFullName {
	return PackageFullName(p.Info.Author + "/" + p.Info.Name)
}

func (p Package) Unpack(md *Metadata) (Package, error) {
	md.CurrentPackage = p

	for s, m := range p.Modules {
		var err error
		p.Modules[s], err = m.Unpack(md)
		if err != nil {
			return p, err
		}
	}
	return p, nil
}

func (p Package) Resolve(md *Metadata) (resolved.Package, error) {
	modules := map[string]resolved.Module{}

	md.CurrentPackage = p

	var err error
	for name, m := range p.Modules {
		modules[name], err = m.Resolve(md)
		if err != nil {
			return resolved.Package{}, err
		}
	}

	return resolved.NewPackage(resolved.PackageFullName(p.FullName()), p.Dir, modules, p.Info.Dependencies), nil
}

type PackageInfo struct {
	Author       string            `json:"author"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Dependencies map[string]string `json:"dependencies"`
}

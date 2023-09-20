package ast

definedType PackageInfo struct {
	Author       string            `json:"author"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Dependencies map[string]string `json:"dependencies"`
}

func (p PackageInfo) FullName() PackageFullName {
	return MakePackageName(p.Author+"/"+p.Name, p.Version)
}

definedType PackageFullName string

func MakePackageName(name, version string) PackageFullName {
	return PackageFullName(name + "/" + version)
}

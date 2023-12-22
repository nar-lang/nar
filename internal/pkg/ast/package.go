package ast

type Package struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	OakVersion   string   `json:"oak-version"`
	Dependencies []string `json:"dependencies"`
	Main         string   `json:"main"`
}

type LoadedPackage struct {
	Urls    map[string]struct{}
	Dir     string
	Package Package
	Sources []string
}

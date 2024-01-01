package ast

type Package struct {
	Name         PackageIdentifier `json:"name"`
	Version      string            `json:"version"`
	NarVersion   string            `json:"nar-version"`
	Dependencies []string          `json:"dependencies"`
	Main         FullIdentifier    `json:"main"`
}

type LoadedPackage struct {
	Urls    map[string]struct{}
	Dir     string
	Package Package
	Sources []string
}

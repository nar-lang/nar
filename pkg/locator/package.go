package locator

type Package interface {
	Info() PackageInfo
	Sources() map[string][]rune
}

type PackageInfo struct {
	Name         string         `json:"name"`
	Version      int            `json:"version"`
	NarVersion   int            `json:"nar-version"`
	Dependencies map[string]int `json:"dependencies"`
	Main         string         `json:"main"`
}

func NewLoadedPackage(info PackageInfo, sources map[string][]rune) Package {
	return loadedPackage{
		info:    info,
		sources: sources,
	}
}

type loadedPackage struct {
	info    PackageInfo
	sources map[string][]rune
}

func (l loadedPackage) Info() PackageInfo {
	return l.info
}

func (l loadedPackage) Sources() map[string][]rune {
	return l.sources
}

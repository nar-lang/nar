package locator

import (
	"os"
	"path/filepath"
)

type Package interface {
	Info() PackageInfo
	SetInfo(info PackageInfo)
	Sources() map[string][]rune
	NativeFilePaths(platform string) ([]string, error)
	Path() string
}

type PackageInfo struct {
	Name         string         `json:"name"`
	Version      int            `json:"version"`
	NarVersion   int            `json:"nar-version"`
	Dependencies map[string]int `json:"dependencies"`
	Main         string         `json:"main"`
}

func NewLoadedPackage(info PackageInfo, sources map[string][]rune, path string) Package {
	return &loadedPackage{
		info:    info,
		sources: sources,
		path:    path,
	}
}

type loadedPackage struct {
	info    PackageInfo
	sources map[string][]rune
	path    string
}

func (l *loadedPackage) Info() PackageInfo {
	return l.info
}

func (l *loadedPackage) SetInfo(info PackageInfo) {
	l.info = info
}

func (l *loadedPackage) Sources() map[string][]rune {
	return l.sources
}

func (l *loadedPackage) NativeFilePaths(platform string) ([]string, error) {
	var result []string
	root := filepath.Join(l.path, "native", platform)
	if _, err := os.Stat(root); err != nil {
		return nil, nil
	}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type() == os.ModeSymlink || d.IsDir() {
			return nil
		}
		result = append(result, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (l *loadedPackage) Path() string {
	return l.path
}

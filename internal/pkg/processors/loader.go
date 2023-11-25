package processors

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-git/go-git/v5"
	"io"
	"oak-compiler/internal/pkg/ast"
	"oak-compiler/internal/pkg/common"
	"os"
	"path/filepath"
	"strings"
)

func LoadPackage(url, cacheDir string, log io.Writer, upgrade bool, loadedPackages []ast.LoadedPackage) []ast.LoadedPackage {
	absBaseDir, err := filepath.Abs(".")
	if err != nil {
		panic(common.SystemError{Message: err.Error()})
	}
	return loadPackage(url, cacheDir, absBaseDir, log, upgrade, loadedPackages)
}

func loadPackage(
	url string, cacheDir string, baseDir string, log io.Writer, upgrade bool, loadedPackages []ast.LoadedPackage,
) []ast.LoadedPackage {
	absPath := filepath.Clean(filepath.Join(baseDir, url))
	loadedPackages, loaded := loadPackageWithPath(url, absPath, cacheDir, log, upgrade, loadedPackages)

	absPath = filepath.Clean(filepath.Join(cacheDir, url))
	if !loaded {
		loadedPackages, loaded = loadPackageWithPath(url, absPath, cacheDir, log, upgrade, loadedPackages)
	}
	if !loaded {
		_, _ = fmt.Fprintf(log, "cloning package `%s`\n", url)
		_, err := git.PlainClone(absPath, false, &git.CloneOptions{
			URL:      fmt.Sprintf("https://%s", url),
			Progress: log,
		})
		if err != nil {
			panic(common.SystemError{Message: err.Error()})
		}
		loadedPackages, loaded = loadPackageWithPath(url, absPath, cacheDir, log, upgrade, loadedPackages)
	} else if upgrade {
		r, err := git.PlainOpen(absPath)
		if err == nil {
			_, _ = fmt.Fprintf(log, "upgrading package... `%s` ", url)
			w, err := r.Worktree()
			if err != nil {
				_, _ = fmt.Fprintf(log, "%s\n", err.Error())
			} else {
				err = w.Pull(&git.PullOptions{
					Progress: log,
				})
				if err != nil {
					_, _ = fmt.Fprintf(log, "%s\n", err.Error())
				} else {
					_, _ = fmt.Fprintf(log, "ok\n")
				}
			}
		}
	}
	if !loaded {
		panic(common.SystemError{Message: "cannot load package `%s`: oak.json file is not found it its root directory"})
	}
	return loadedPackages
}

func loadPackageWithPath(
	url string, absPath string, cacheDir string, log io.Writer, upgrade bool, loadedPackage []ast.LoadedPackage,
) ([]ast.LoadedPackage, bool) {
	packageFilePath := filepath.Join(absPath, "oak.json")
	fileData, err := os.ReadFile(packageFilePath)

	if errors.Is(err, os.ErrNotExist) {
		return loadedPackage, false
	}

	if err != nil {
		panic(common.SystemError{
			Message: fmt.Sprintf("failed to read package `%s` descriptor: %s", url, err.Error()),
		})
	}

	var pkg ast.Package
	err = json.Unmarshal(fileData, &pkg)
	if err != nil {
		panic(common.SystemError{
			Message: fmt.Sprintf("failed to parse package `%s` descriptor file: %s", url, err.Error()),
		})
	}

	for i, loaded := range loadedPackage {
		if loaded.Package.Name == pkg.Name {
			if loaded.Package.Version != pkg.Version {
				if loaded.Package.Version > pkg.Version {
					_, _ = fmt.Fprintf(log,
						"package `%s` version collision %s vs %s, using higher version",
						pkg.Name, pkg.Version, pkg.Version)
				}
			}
			loadedPackage = append(loadedPackage[:i], loadedPackage[i+1:]...)
			if loaded.Package.Version >= pkg.Version { //move loaded to the end
				loadedPackage = append(loadedPackage, loaded)
				return loadedPackage, true
			}
			if loaded.Package.Version < pkg.Version { //remove package with lower version
				break
			}
		}
	}

	src, err := readDir(filepath.Join(absPath, "src"), ".oak", nil)
	if err != nil {
		panic(common.SystemError{
			Message: fmt.Sprintf("failed to read package `%s` sources: %s", url, err.Error()),
		})
	}

	loadedPackage = append(loadedPackage, ast.LoadedPackage{Url: url, Dir: absPath, Package: pkg, Sources: src})

	for _, depUrl := range pkg.Dependencies {
		loadedPackage = loadPackage(depUrl, cacheDir, absPath, log, upgrade, loadedPackage)
	}

	return loadedPackage, true
}

func readDir(path, ext string, files []string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.IsDir() {
			files, err = readDir(path, ext, files)
			if err != nil {
				return nil, err
			}
		} else if strings.EqualFold(filepath.Ext(e.Name()), ext) {
			files = append(files, filepath.Join(path, e.Name()))
		}
	}

	return files, nil
}

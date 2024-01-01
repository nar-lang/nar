package processors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-git/go-git/v5"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type Progress func(value float32, message string)

func LoadPackage(
	url, cacheDir string, baseDir string, progress Progress, upgrade bool,
	loadedPackages map[ast.PackageIdentifier]*ast.LoadedPackage,
) (*ast.LoadedPackage, error) {
	if baseDir != "" {
		var err error
		baseDir, err = filepath.Abs(baseDir)
		if err != nil {
			return nil, common.NewSystemError(err)
		}
	}
	return loadPackage(url, cacheDir, baseDir, progress, upgrade, loadedPackages)
}

func loadPackage(
	url string, cacheDir string, baseDir string, progress Progress, upgrade bool,
	loadedPackages map[ast.PackageIdentifier]*ast.LoadedPackage,
) (*ast.LoadedPackage, error) {
	absPath := filepath.Clean(url)
	if baseDir != "" {
		absPath = filepath.Clean(filepath.Join(baseDir, url))
	}
	loaded, err := loadPackageWithPath(url, absPath, cacheDir, progress, upgrade, loadedPackages)
	if err != nil {
		return nil, err
	}

	absPath = filepath.Clean(filepath.Join(cacheDir, url))
	if loaded == nil {
		loaded, err = loadPackageWithPath(url, absPath, cacheDir, progress, upgrade, loadedPackages)
		if err != nil {
			return nil, err
		}
	}
	if loaded == nil {
		progress(0, fmt.Sprintf("downloading package `%s`", url))
		w := bytes.NewBufferString("")
		_, err := git.PlainClone(absPath, false, &git.CloneOptions{
			URL:      fmt.Sprintf("https://%s", url),
			Progress: w,
		})
		if err != nil {
			return nil, common.NewSystemError(fmt.Errorf("%s\n%w", w.String(), err))
		} else {
			progress(1, fmt.Sprintf("%s\npackage `%s` downloaded", url, w.String()))
		}
		loaded, err = loadPackageWithPath(url, absPath, cacheDir, progress, upgrade, loadedPackages)
		if err != nil {
			return nil, err
		}
	} else if upgrade {
		r, err := git.PlainOpen(absPath)
		if err == nil {
			worktree, err := r.Worktree()
			if err != nil {
				return nil, common.NewSystemError(fmt.Errorf("failed to update package `%s`\n%w", url, err))
			} else {
				w := bytes.NewBufferString("")
				err = worktree.Pull(&git.PullOptions{
					Progress: w,
				})
				if err != nil {
					return nil, common.NewSystemError(
						fmt.Errorf("failed to update package `%s`\n%w\n%s", url, err, w.String()))
				} else {
					progress(1, fmt.Sprintf("%s\npackage `%s` updated", url, w.String()))
				}
			}
		}
	}
	return loaded, nil
}

func loadPackageWithPath(
	url string, absPath string, cacheDir string, progress Progress, upgrade bool,
	loadedPackages map[ast.PackageIdentifier]*ast.LoadedPackage,
) (*ast.LoadedPackage, error) {
	packageFilePath := filepath.Join(absPath, "nar.json")
	fileData, err := os.ReadFile(packageFilePath)
	var loaded *ast.LoadedPackage

	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, common.NewSystemError(fmt.Errorf("failed to read package `%s` descriptor: %w", url, err))
	}

	var pkg ast.Package
	err = json.Unmarshal(fileData, &pkg)
	if err != nil {
		return nil, common.NewSystemError(
			fmt.Errorf("failed to parse package `%s` descriptor file: %w", url, err))
	}

	insert := false
	var ok bool
	if loaded, ok = loadedPackages[pkg.Name]; ok {
		if pkg.Version < loaded.Package.Version {
			progress(0.5, fmt.Sprintf(
				"package `%s` version collision %s vs %s, using higher version",
				pkg.Name, pkg.Version, pkg.Version))
		} else if loaded.Package.Version > pkg.Version {
			progress(0.5, fmt.Sprintf(
				"package `%s` version collision %s vs %s, using higher version",
				pkg.Name, pkg.Version, pkg.Version))
			insert = true
		} else if loaded.Package.Version == pkg.Version {
			loaded.Urls[url] = struct{}{}
		}
	} else {
		insert = true
	}

	if insert {
		src, err := readDir(filepath.Join(absPath, "src"), ".nar", nil)
		if err != nil {
			return nil, common.NewSystemError(fmt.Errorf(
				"failed to read package `%s` sources: %w", url, err))
		}

		slices.Sort(src)
		loaded = &ast.LoadedPackage{
			Urls:    map[string]struct{}{url: {}},
			Dir:     absPath,
			Package: pkg,
			Sources: src,
		}

		loadedPackages[pkg.Name] = loaded

		for _, depUrl := range pkg.Dependencies {
			_, err = loadPackage(depUrl, cacheDir, absPath, progress, upgrade, loadedPackages)
			if err != nil {
				return nil, err
			}
		}
	}

	return loaded, nil
}

func readDir(path, ext string, files []string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.IsDir() {
			files, err = readDir(filepath.Join(path, e.Name()), ext, files)
			if err != nil {
				return nil, err
			}
		} else if strings.EqualFold(filepath.Ext(e.Name()), ext) {
			files = append(files, filepath.Join(path, e.Name()))
		}
	}

	return files, nil
}

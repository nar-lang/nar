package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/exp/slices"
	"oak-compiler/pkg/ast"
	"oak-compiler/pkg/parsed"
	"os"
	"path"
	"strings"
)

definedType ParseOptions struct {
	CacheDirectory   string
	DownloadPackages bool
}

func ParsePackagesFromFs(opts ParseOptions, packageDirs ...string) (ParsedPackages, error) {
	var sources []PackageSource
	for _, packageDir := range packageDirs {

		if _, err := os.Stat(path.Join(packageDir, "oak.json")); errors.Is(err, os.ErrNotExist) {
			packageDir = path.Join(opts.CacheDirectory, packageDir)
		}

		src, err := loadPackageFromFs(packageDir)
		if err != nil {
			return nil, err
		}
		sources = append(sources, src)
	}
	return ParsePackageFromSources(opts, sources...)
}

func ParsePackageFromSources(opts ParseOptions, source ...PackageSource) (ParsedPackages, error) {
	sourceMap := map[ast.PackageFullName]PackageSource{}

	for _, pkg := range source {
		sourceMap[pkg.Info.FullName()] = pkg
	}

	depLoaded := true
	for depLoaded {
		depLoaded = false

		for _, src := range sourceMap {
			var deps []ast.PackageFullName
			for depName, depVersion := range src.Info.Dependencies {
				deps = append(deps, ast.MakePackageName(depName, depVersion))
			}
			slices.Sort(deps)

			for _, fullName := range deps {
				if _, loaded := sourceMap[fullName]; loaded {
					continue
				}

				depLoaded = false
				pkgDir := path.Join(opts.CacheDirectory, string(fullName))
				_, err := os.Stat(path.Join(pkgDir, "oak.json"))
				if err == nil {
					sourceMap[fullName], err = loadPackageFromFs(pkgDir)
					if err != nil {
						return nil, err
					}
					continue
				}

				if opts.DownloadPackages {
					panic("downloading not implemented yet")
					continue
				}

				return nil, fmt.Errorf("cannot resolve or download package `%s`", fullName)
			}
		}
	}

	parsedMap := map[ast.PackageFullName]parsed.Package{}
	for name, pkgSrc := range sourceMap {
		var err error
		parsedMap[name], err = parsePackage(pkgSrc)
		if err != nil {
			return nil, err
		}
	}

	return parsedMap, nil
}

func parsePackage(pkgSource PackageSource) (parsed.Package, error) {
	parsedPackage := parsed.Package{
		Info:    pkgSource.Info,
		Modules: map[string]parsed.Module{},
	}

	for fileName, src := range pkgSource.Modules {
		parsedModule, err := parseModule(fileName, src, pkgSource.Info.FullName())
		if err != nil {
			return parsed.Package{}, err
		}
		if _, exists := parsedPackage.Modules[parsedModule.Name()]; exists {
			return parsed.Package{},
				fmt.Errorf("duplicate module name `%s` in package `%s`", parsedModule.Name(), pkgSource.Info.FullName())
		}
		parsedPackage.Modules[parsedModule.Name()] = parsedModule
	}

	return parsedPackage, nil
}

func loadPackageFromFs(packageDir string) (PackageSource, error) {
	pkg := PackageSource{
		Modules: map[ModuleFileName]ModuleSource{},
	}
	oakJsonPath := path.Join(packageDir, "oak.json")
	oakJson, err := os.ReadFile(oakJsonPath)
	if err != nil {
		return PackageSource{}, fmt.Errorf("failed to load package file. %w", err)
	}

	err = json.Unmarshal(oakJson, &pkg.Info)
	if err != nil {
		return PackageSource{}, fmt.Errorf("failed to parse `%s`: %w", oakJsonPath, err)
	}

	sourcesDir := path.Join(packageDir, "src")
	sources, err := getSourceFileNames(sourcesDir)
	if err != nil {
		return PackageSource{}, err
	}

	for _, sourcePath := range sources {
		src, err := os.ReadFile(sourcePath)
		if err != nil {
			return PackageSource{}, fmt.Errorf("failed to read file `%s`: %w", sourcePath, err)
		}
		pkg.Modules[ModuleFileName(sourcePath)] = ModuleSource(src)
	}

	return pkg, nil
}

func getSourceFileNames(dirPath string) ([]string, error) {
	fs, err := os.ReadDir(dirPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read dir `%s`:\n%w", dirPath, err)
	}
	var result []string
	for _, fi := range fs {
		if fi.IsDir() {
			nested, nerr := getSourceFileNames(path.Join(dirPath, fi.Name()))
			if nerr != nil {
				return nil, nerr
			}
			result = append(result, nested...)
		} else if strings.ToLower(path.Ext(fi.Name())) == ".oak" {
			result = append(result, path.Join(dirPath, fi.Name()))
		}
	}

	return result, nil
}

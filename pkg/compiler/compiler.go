package compiler

import (
	"encoding/json"
	"fmt"
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/parsed"
	"oak-compiler/pkg/resolved"
	"os"
	"path"
	"strings"
)

func Compile(inDir string, outDir string, mainName string) error {
	md := parsed.NewMetadata()

	pkg, err := loadPackageWithDependencies(inDir, md.Packages)
	if err != nil {
		return err
	}

	var resolvedPackages []resolved.Package

	for n, pkg := range md.Packages {
		md.Packages[n], err = pkg.Unpack(&md)
		if err != nil {
			return err
		}
	}

	for _, pkg := range md.Packages {
		rp, err := pkg.Resolve(&md)
		if err != nil {
			return err
		}
		resolvedPackages = append(resolvedPackages, rp)
	}

	for _, pkg := range resolvedPackages {
		err := pkg.Write(outDir)
		if err != nil {
			return err
		}
	}

	if mainName != "" {
		resolved.PackageFullName(pkg.FullName()).SafeName()
		modDir := path.Join(outDir, string(pkg.FullName()), "_main")
		safeName := resolved.PackageFullName(pkg.FullName()).SafeName()
		err := os.MkdirAll(modDir, 0777)
		if err != nil {
			return err
		}
		sb := fmt.Sprintf(
			"package main\n\n"+
				"import (\n"+
				"\t%s \"%s\"\n"+
				"\t\"os\"\n"+
				")\n\n"+
				"func main() {\n"+
				"\tos.Exit(%s.%s()())\n"+
				"}\n",
			safeName,
			pkg.FullName(),
			safeName,
			strings.ReplaceAll(mainName, ".", "_"),
		)
		err = os.WriteFile(path.Join(modDir, "main.go"), []byte(sb), 0666)
		if err != nil {
			return err
		}
	}

	return nil
}

func loadPackageWithDependencies(packageUrl string, packages map[parsed.PackageFullName]parsed.Package) (parsed.Package, error) {
	pkg, err := loadPackage(packageUrl)
	if err != nil {
		return parsed.Package{}, err
	}

	packages[parsed.PackageFullName(string(pkg.FullName()))] = pkg
	for depPackage, version := range pkg.Info.Dependencies {
		if loadedPackage, ok := packages[parsed.PackageFullName(depPackage)]; ok {
			if loadedPackage.Info.Version != version {
				return parsed.Package{},
					fmt.Errorf("dependency version collision for package %s:\n"+
						"package %s requested version %s but already loaded with version %s",
						pkg.FullName(), depPackage, version, loadedPackage.Info.Version)
			}
		} else {
			depPackageUrl := depPackage
			if strings.HasPrefix(version, ".") {
				depPackageUrl = path.Clean(path.Join(packageUrl, version)) //load from filesystem
			}
			_, err = loadPackageWithDependencies(depPackageUrl, packages)
			p := packages[parsed.PackageFullName(depPackage)]
			p.Info.Version = version
			packages[parsed.PackageFullName(depPackage)] = p
			if err != nil {
				return parsed.Package{}, err
			}
		}
	}
	return pkg, nil
}

func loadPackage(packageDir string) (parsed.Package, error) {
	pkg := parsed.Package{
		Dir:     packageDir,
		Modules: map[string]parsed.Module{},
	}
	oakJsonPath := path.Join(packageDir, "oak.json")
	oakJson, err := os.ReadFile(oakJsonPath)
	if err != nil {
		return pkg, fmt.Errorf("failed to load package file. %w", err)
	}

	err = json.Unmarshal(oakJson, &pkg.Info)
	if err != nil {
		return pkg, fmt.Errorf("failed to parse `%s`: %w", oakJsonPath, err)
	}

	sourcesDir := path.Join(packageDir, "src")
	sources, err := getSourceFileNames(sourcesDir)
	if err != nil {
		return pkg, err
	}

	for _, sourcePath := range sources {
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			return pkg, fmt.Errorf("failed to read file `%s`: %w", sourcePath, err)
		}
		cursor := misc.NewCursor(sourcePath, []rune(string(data)))
		module, err := parsed.ParseModule(&cursor, pkg.FullName())
		if err != nil {
			return pkg, err
		}
		if _, ok := pkg.Modules[module.Name()]; ok {
			return pkg, fmt.Errorf("package `%s` has redeclared module `%s`",
				packageDir, module.Name())
		}
		pkg.Modules[module.Name()] = module
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

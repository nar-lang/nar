package main

import (
	"flag"
	"fmt"
	"oak-compiler/pkg/parser"
	"os"
	"strings"
)

func main() {
	err := mainWithError()
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
}

func mainWithError() error {
	inDir := flag.String("packages", "", "package directories to compile, separated with `;`")
	cache := flag.String("cache", "./oak-cache", "cache directory")
	offline := flag.Bool("offline", false, "disable packages downloading")

	flag.Parse()

	if *inDir == "" {
		flag.Usage()
		return fmt.Errorf("no input package provided")
	}

	dirs := strings.Split(*inDir, ";")

	packages, err := parser.ParsePackagesFromFs(
		parser.ParseOptions{
			CacheDirectory:   *cache,
			DownloadPackages: !*offline,
		},
		dirs...,
	)
	if err != nil {
		return err
	}

	_, err = packages.Compile()
	if err != nil {
		return err
	}

	return nil
}

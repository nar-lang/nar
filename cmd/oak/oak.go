package main

import (
	"flag"
	"log"
	oakc "oak-compiler/pkg"
	"os"
	"path/filepath"
)

func main() {
	homeDir, _ := os.UserHomeDir()
	defaultCacheDir := filepath.Join(homeDir, ".oak")

	out := flag.String("out", "out.acorn", "output file path")
	release := flag.Bool("release", false, "strip debug symbols")
	cacheDir := flag.String("cache", defaultCacheDir, "package cache directory")
	upgrade := flag.Bool("upgrade", false, "upgrade packages")
	flag.Parse()

	err := oakc.Compile(flag.Args(), *out, !*release, *upgrade, *cacheDir, os.Stdout)
	if err != nil {
		log.Fatal(err.Error())
	}
}

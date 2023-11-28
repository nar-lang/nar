package main

import (
	"flag"
	"log"
	oakc "oak-compiler/pkg"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	homeDir, _ := os.UserHomeDir()
	defaultCacheDir := filepath.Join(homeDir, ".oak")

	out := flag.String("out", "build", "output file path (or directory if linking)")
	release := flag.Bool("release", false, "strip debug symbols")
	cacheDir := flag.String("cache", defaultCacheDir, "package cache directory")
	upgrade := flag.Bool("upgrade", false, "upgrade packages")
	link := flag.String("link", "", "link program for specific platform (available: js)")
	pack := flag.String("pack", "", "command to pack resulted executable.\n"+
		"  examples\n"+"  js: `webpack-cli --entry build/index.source.js -o ./build`")
	noClean := flag.Bool("no-clean", false, "don't clean up intermediate artifacts after packing")
	flag.Parse()

	outStream := os.Stdout

	linker := oakc.GetLinker(*link)
	err, loadedPackages := oakc.Compile(
		flag.Args(), linker.GetOutFileLocation(*out),
		!*release, *upgrade, *cacheDir, outStream)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = linker.Link(loadedPackages[0].Package.Main, loadedPackages, *out, !*release, *upgrade, *cacheDir, outStream)
	if err != nil {
		log.Fatal(err.Error())
	}
	if *pack != "" {
		args := splitArgs(*pack)
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = outStream
		cmd.Stderr = outStream
		err := cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
	if !*noClean {
		linker.Cleanup()
	}
}

// splitArgs splits string to words but keeps quoted strings as one word
func splitArgs(s string) []string {
	var args []string
	var inQuotes bool
	var currentArg string
	for _, c := range s {
		if c == '"' {
			inQuotes = !inQuotes
			continue
		}
		if c == ' ' && !inQuotes {
			args = append(args, currentArg)
			currentArg = ""
			continue
		}
		currentArg += string(c)
	}
	if currentArg != "" {
		args = append(args, currentArg)
	}
	return args
}

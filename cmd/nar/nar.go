package main

import (
	"flag"
	"fmt"
	"nar-compiler/internal/pkg/lsp"
	"nar-compiler/internal/pkg/processors"
	narc "nar-compiler/pkg"
	"nar-compiler/pkg/bytecode"
	"nar-compiler/pkg/locator"
	"nar-compiler/pkg/logger"
	"os"
	"path/filepath"
)

func main() {
	homeDir, _ := os.UserHomeDir()
	defaultCacheDir := filepath.Join(homeDir, ".nar")

	//out := flag.String("out", "build", "output file path (or directory if linking)")
	release := flag.Bool("release", false, "strip debug symbols")
	cache := flag.String("cache", defaultCacheDir, "package cache directory")
	//upgrade := flag.Bool("upgrade", false, "upgrade packages")
	//link := flag.String("link", "", "link program for specific platform (available: js)")
	//pack := flag.String("pack", "", "command to pack resulted executable.\n"+ "  examples\n"+"  js: `webpack-cli --entry build/index.source.js -o ./build`")
	//noClean := flag.Bool("no-clean", false, "don't clean up intermediate artifacts after packing")
	runLsp := flag.String("lsp", "", "start language server with given transport (stdio/tcp)")
	lspPort := flag.Int("lsp-port", 0, "port for tcp transport")
	showVersion := flag.Bool("version", false, "show version")
	run := flag.Bool("run", false, "execute the compiled program")
	//runBinar := flag.String("run-binar", "", "execute specified compiled program")
	flag.Parse()

	if *showVersion {
		doShowVersion()
		return
	}

	if *runLsp != "" {
		doLsp(*runLsp, *lspPort, *cache)
		return
	}

	bin := doCompile(*release, *cache, flag.Args())

	if bin != nil && *run {
		doRun(bin)
	}
}

func doCompile(release bool, cacheDir string, packages []string) *bytecode.Binary {
	log := &logger.LogWriter{FailOnErr: true}

	var providers []locator.Provider
	for _, path := range packages {
		providers = append(providers, locator.NewFileSystemPackageProvider(path))
	}
	if cacheDir != "" {
		providers = append(providers, locator.NewDirectoryProvider(cacheDir))
	}
	//TODO: add git repository provider
	var lc = locator.NewLocator(providers...)

	bin := narc.Compile(log, lc, !release)
	if !log.Err() {
		doLink(log, lc, bin)
	}

	log.Flush(os.Stdout)
	return bin
}

func doLink(log *logger.LogWriter, loadedPackages locator.Locator, entry *bytecode.Binary) {
	/*if !logger.Err() {
		if len(logger.Errors()) == 0 {
			err := linker.Link(entry, loadedPackages, out, !release, upgrade, cacheDir, logger)
			if err != nil {
				logger.Err(err)
			} else {
				if pack != "" {
					w := bytes.NewBufferString("")
					args := splitArgs(pack)
					cmd := exec.Command(args[0], args[1:]...)
					cmd.Stdout = w
					cmd.Stderr = w
					err := cmd.Run()
					logger.Trace(w.String())
					if err != nil {
						logger.Err(err)
					}
				}
				if !noClean {
					linker.Cleanup()
				}
			}
		}
	}
	*/

}

func doLsp(runLsp string, lspPort int, cacheDir string) {
	log := &logger.LogWriter{FailOnErr: true}
	err := narc.LanguageServer(runLsp, lspPort, cacheDir)
	if err != nil {
		log.Err(err)
	}
	log.Flush(os.Stdout)
}

func doShowVersion() {
	fmt.Printf("nar compiler version: %s\n"+
		"language server protocol version: %s\n"+
		"binar format version: %d.%02d\n",
		processors.Version, lsp.Version,
		bytecode.BinaryFormatVersion/100,
		bytecode.BinaryFormatVersion%100)
}

func doRun(bin *bytecode.Binary) {

}

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

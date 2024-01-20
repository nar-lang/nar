package main

import (
	"bytes"
	"flag"
	"fmt"
	"nar-compiler/internal/pkg/common"
	"nar-compiler/internal/pkg/lsp"
	"nar-compiler/internal/pkg/processors"
	narc "nar-compiler/pkg"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	homeDir, _ := os.UserHomeDir()
	defaultCacheDir := filepath.Join(homeDir, ".nar")

	out := flag.String("out", "build", "output file path (or directory if linking)")
	release := flag.Bool("release", false, "strip debug symbols")
	cacheDir := flag.String("cache", defaultCacheDir, "package cache directory")
	upgrade := flag.Bool("upgrade", false, "upgrade packages")
	link := flag.String("link", "", "link program for specific platform (available: js)")
	pack := flag.String("pack", "", "command to pack resulted executable.\n"+
		"  examples\n"+"  js: `webpack-cli --entry build/index.source.js -o ./build`")
	noClean := flag.Bool("no-clean", false, "don't clean up intermediate artifacts after packing")
	runLsp := flag.String("lsp", "", "start language server with given transport (stdio/tcp)")
	lspPort := flag.Int("lsp-port", 0, "port for tcp transport")
	showVersion := flag.Bool("version", false, "show version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("nar compiler version: %s\nlanguage server protocol version: %s\n", processors.Version, lsp.Version)
		return
	}

	log := &common.LogWriter{}

	if *runLsp != "" {
		err := narc.LanguageServer(*runLsp, *lspPort, *cacheDir)
		if err != nil {
			log.Err(err)
		}
		return
	}

	if len(flag.Args()) == 0 {
		log.Err(common.NewSystemError(fmt.Errorf("no input packages, run compiler as `nar <path-to-package>`")))
	} else {

		linker := narc.GetLinker(*link)

		loadedPackages, entry := narc.Compile(
			flag.Args(), linker.GetOutFileLocation(*out),
			!*release, *upgrade, *cacheDir, log)
		if !log.HasErrors() {
			if len(log.Errors()) == 0 {
				err := linker.Link(entry, loadedPackages, *out, !*release, *upgrade, *cacheDir, log)
				if err != nil {
					log.Err(err)
				} else {
					if *pack != "" {
						w := bytes.NewBufferString("")
						args := splitArgs(*pack)
						cmd := exec.Command(args[0], args[1:]...)
						cmd.Stdout = w
						cmd.Stderr = w
						err := cmd.Run()
						log.Trace(w.String())
						if err != nil {
							log.Err(err)
						}
					}
					if !*noClean {
						linker.Cleanup()
					}
				}
			}
		}
	}
	log.Flush(os.Stdout)
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

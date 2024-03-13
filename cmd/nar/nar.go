package main

import (
	"bytes"
	"flag"
	"fmt"
	"nar-compiler/pkg/bytecode"
	"nar-compiler/pkg/compiler"
	"nar-compiler/pkg/linker"
	"nar-compiler/pkg/locator"
	"nar-compiler/pkg/logger"
	"nar-compiler/pkg/lsp"
	"nar-compiler/pkg/runtime"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	homeDir, _ := os.UserHomeDir()
	cache := flag.String("cache", filepath.Join(homeDir, ".nar"), "package cache directory")
	release := flag.Bool("release", false, "strip debug symbols")
	link := flag.String("link", "dll", "link program for specific platform (available: dll)")
	out := flag.String("out", "program.binar", "output file name")
	runLsp := flag.String("lsp", "", "start language server with given transport (stdio/tcp)")
	lspPort := flag.Int("lsp-port", 0, "port for tcp transport")
	showVersion := flag.Bool("version", false, "show version")
	run := flag.Bool("run", false, "execute program after compilation")
	binar := flag.String("binar", "", "execute program from binar file")
	flag.Parse()

	if *showVersion {
		doShowVersion()
		return
	}

	if *runLsp != "" {
		doLsp(*runLsp, *lspPort, *cache)
		return
	}

	if *binar != "" {
		if err := doRunBinar(*binar); err != nil {
			fmt.Println(err)
		}
		return
	}

	var lnk linker.Linker
	switch *link {
	case "dll":
		lnk = linker.NewDllLinker(*out)
	}

	bin := doCompile(*release, *cache, lnk, flag.Args())

	if bin != nil && *run {
		doRun(bin, filepath.Dir(*out))
	}
}

func doRunBinar(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	binary, err := bytecode.Read(bytes.NewReader(data))
	if err != nil {
		return err
	}
	return doRun(binary, filepath.Dir(path))
}

func doCompile(release bool, cacheDir string, link linker.Linker, packages []string) *bytecode.Binary {
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

	bin := compiler.Compile(log, lc, link, !release)
	log.Trace("compilation finished")
	log.Flush(os.Stdout)
	return bin
}

func doLsp(runLsp string, lspPort int, cacheDir string) {
	log := &logger.LogWriter{FailOnErr: true}
	err := lsp.LanguageServer(runLsp, lspPort, cacheDir)
	if err != nil {
		log.Err(err)
	}
	log.Flush(os.Stdout)
}

func doShowVersion() {
	vts := func(v int) string { return fmt.Sprintf("%d.%02d", v/100, v%100) }
	fmt.Printf("nar compiler version: %s\n"+
		"language server protocol version: %s\n"+
		"binar format version: %s\n",
		vts(compiler.Version()),
		vts(lsp.Version()),
		vts(bytecode.Version()))
}

func doRun(bin *bytecode.Binary, libsPath string) (err error) {
	rt := runtime.NewRuntime(bin)
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v\n%s", r, strings.Join(rt.Stack(), "\n"))
		}
	}()
	for name, version := range bin.Packages {
		err = runtime.RegisterNativeLibrary(rt, string(name), int(version), libsPath)
		if err != nil {
			return
		}
	}
	if bin.Entry == "" {
		err = fmt.Errorf("entry point not found")
		return
	}
	_, err = rt.Apply(bin.Entry)
	return
}

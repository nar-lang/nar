package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/nar-lang/nar-common/bytecode"
	"github.com/nar-lang/nar-common/logger"
	"github.com/nar-lang/nar-compiler/compiler"
	"github.com/nar-lang/nar-compiler/linker"
	"github.com/nar-lang/nar-compiler/locator"
	"github.com/nar-lang/nar-lsp"
	"github.com/nar-lang/nar-runtime/runtime"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	println(strings.Join(os.Args, " "))
	homeDir, _ := os.UserHomeDir()
	cache := flag.String("cache", filepath.Join(homeDir, ".nar", "packages"), "package cache directory")
	release := flag.Bool("release", false, "strip debug symbols")
	link := flag.String("link", "dll", "link program for specific platform (available: dll)")
	out := flag.String("out", "program.binar", "output file name")
	showVersion := flag.Bool("version", false, "show version")
	run := flag.Bool("run", false, "execute program after compilation")
	binar := flag.String("binar", "", "execute program from binar file")
	lspEnable := flag.Bool("lsp", false, "start language server")
	flag.Bool("stdio", false, "use stdio for language server (default)")
	lspTcp := flag.Int("tcp", 0, "use tcp transport with given port for language server")
	flag.Parse()

	if *showVersion {
		doShowVersion()
		return
	}

	if *lspEnable {
		doLsp(*lspTcp, *cache)
		return
	}

	if *binar != "" {
		if err := doRunBinar(*binar); err != nil {
			fmt.Println(err)
			os.Exit(-1)
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
		err := doRun(bin, filepath.Dir(*out))
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
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

func doLsp(tcpPort int, cacheDir string) {
	log := &logger.LogWriter{FailOnErr: true}
	err := nar_lsp.LanguageServer(tcpPort, cacheDir)
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
		vts(nar_lsp.Version()),
		vts(bytecode.Version()))
}

func doRun(bin *bytecode.Binary, libsPath string) (err error) {
	rt, err := runtime.NewRuntime(bin, libsPath)
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("runtime error: %v\nat\n%s", r, strings.Join(rt.Stack(), "\n"))
		}
	}()
	if err != nil {
		return

	}
	if bin.Entry == "" {
		err = fmt.Errorf("entry point not found")
		return
	}
	_, err = rt.Apply(bin.Entry)
	rt.Destroy()
	return
}

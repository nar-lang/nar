package main

/*
#cgo CFLAGS: -I${SRCDIR}/../../../nar-runtime-c/include
#cgo LDFLAGS: -ldl -L${SRCDIR}/../../../nar-runtime-c/include -lnar-runtime-c
#include <string.h>
#include <nar.h>
#include <nar-runtime.h>
*/
import "C"
import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/nar-lang/nar-compiler/bytecode"
	"github.com/nar-lang/nar-compiler/compiler"
	"github.com/nar-lang/nar-compiler/linker"
	"github.com/nar-lang/nar-compiler/locator"
	"github.com/nar-lang/nar-compiler/logger"
	"github.com/nar/pkg"
	"os"
	"path/filepath"
	"strings"
	"unsafe"
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
		buf := bytes.NewBuffer(nil)
		w := bufio.NewWriter(buf)
		err := bin.Write(w, true)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
		err = w.Flush()
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}

		err = doRun(buf.Bytes(), filepath.Dir(*out))
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
	return doRun(data, filepath.Dir(path))
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
	err := pkg.LanguageServer(tcpPort, cacheDir)
	if err != nil {
		log.Err(err)
	}
	log.Flush(os.Stdout)
}

func doShowVersion() {
	vts := func(v uint32) string { return fmt.Sprintf("%d.%02d", v/100, v%100) }
	fmt.Printf("nar compiler version: %s\n"+
		"language server protocol version: %s\n"+
		"binar format version: %s\n",
		vts(compiler.Version),
		vts(pkg.Version),
		vts(bytecode.Version))
}

func doRun(data []byte, libsPath string) (err error) {
	to_str := func(s C.nar_cstring_t) string {
		sz := C.strlen(s)
		return string(C.GoBytes(unsafe.Pointer(s), C.int(sz)))
	}

	var btc C.nar_bytecode_t
	buf := C.CBytes(data)
	bytecodeLoadResult := C.nar_bytecode_new(
		C.nar_size_t(len(data)),
		(*C.nar_byte_t)(buf),
		&btc)
	C.free(buf)
	buf = nil

	var rt C.nar_runtime_t = nil
	var entryPoint C.nar_cstring_t = nil
	var result C.nar_object_t = C.INVALID_OBJECT

	if bytecodeLoadResult != 0 {
		err = fmt.Errorf("could not create bytecode (error code %d)", int(bytecodeLoadResult))
		goto cleanup
	}

	buf = C.CBytes(append([]byte(libsPath), 0))
	rt = C.nar_runtime_new(btc, C.nar_cstring_t(buf))
	if C.nar_get_last_error(rt) != nil {
		err = fmt.Errorf("could not create runtime (error message: %s)", to_str(C.nar_get_last_error(rt)))
		goto cleanup
	}
	C.free(buf)
	buf = nil

	entryPoint = C.nar_bytecode_get_entry(btc)

	result = C.nar_apply(rt, entryPoint, 0, nil)
	if C.nar_object_is_valid(rt, result) == 0 {
		err = fmt.Errorf("could not execute entry point %s (error message: %s)",
			to_str(entryPoint),
			to_str(C.nar_get_last_error(rt)))
		goto cleanup
	}

cleanup:
	C.nar_runtime_free(rt)
	C.nar_bytecode_free(btc)
	C.free(buf)
	return err
}

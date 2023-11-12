package main

import (
	"flag"
	"fmt"
	"oak-compiler/ast"
	"oak-compiler/ast/bytecode"
	"oak-compiler/ast/normalized"
	"oak-compiler/ast/parsed"
	"oak-compiler/ast/typed"
	"oak-compiler/common"
	"oak-compiler/processors"
	"os"
)

func main() {
	out := flag.String("out", "out.acorn", "output file path")
	release := flag.Bool("release", false, "strip debug symbols")
	flag.Parse()

	errorChecked(flag.Args(), *out, !*release)
}

func errorChecked(ins []string, out string, debug bool) {
	defer func() {
		x := recover()
		switch x.(type) {
		case common.Error:
			{
				e := x.(common.Error)
				line, col := e.Location.GetLineAndColumn()
				fmt.Printf("%s:%d:%d %s\n", e.Location.FilePath, line, col, e.Message)
				return
			}
		case common.SystemError:
			{
				e := x.(common.SystemError)
				fmt.Println(e.Message)
				break
			}
		}
		if x != nil {
			panic(x)
		}
	}()

	parsedModules := map[string]parsed.Module{}
	normalizedModules := map[string]normalized.Module{}
	typedModules := map[string]*typed.Module{}
	bin := bytecode.Binary{
		Exports:   map[ast.ExternalIdentifier]bytecode.Pointer{},
		FuncsMap:  map[ast.PathIdentifier]bytecode.Pointer{},
		StringMap: map[string]bytecode.StringHash{},
		ConstMap:  map[bytecode.PackedConst]bytecode.ConstHash{},
	}

	for _, arg := range ins {
		path := processors.Parse(arg, parsedModules)
		processors.Normalize(path, parsedModules, normalizedModules)
		processors.CheckTypes(path, normalizedModules, typedModules)
		processors.Compile(path, typedModules, &bin)
	}

	file, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		panic(common.SystemError{Message: err.Error()})
	}

	bin.Build(file, debug)
	println("huge success")
}

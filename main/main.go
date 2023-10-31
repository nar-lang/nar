package main

import (
	"flag"
	"fmt"
	"oak-compiler/ast"
	"oak-compiler/ast/normalized"
	"oak-compiler/ast/parsed"
	"oak-compiler/ast/typed"
	"oak-compiler/common"
	"oak-compiler/processors"
	"os"
)

func main() {
	flag.Parse()

	errorChecked()
}

func errorChecked() {
	defer func() {
		x := recover()
		switch x.(type) {
		case common.Error:
			{
				e := x.(common.Error)
				line, col := getErrorLine(e.Location)
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

	for _, arg := range flag.Args() {
		path := processors.Parse(arg, parsedModules)
		processors.Normalize(path, parsedModules, normalizedModules)
		processors.CheckTypes(path, normalizedModules, typedModules)
	}
	println("huge success")
}

func getErrorLine(loc ast.Location) (line int, column int) {
	data, _ := os.ReadFile(loc.FilePath)
	text := []rune(string(data))

	line = 1
	column = 1

	for i := uint64(0); i < loc.Position; i++ {
		if '\n' == text[i] {
			line++
			column = 1
		} else {
			column++
		}
	}
	return
}

package main

import (
	"flag"
	"fmt"
	"oak-compiler/pkg/compiler"
	"os"
)

func main() {
	err := mainWithError()
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
}

func mainWithError() error {
	outDir := flag.String("out", "build", "output directory")
	inDir := flag.String("in", "", "package directory to compile")
	mainName := flag.String("main", "", "main function to compile executable")

	flag.Parse()

	if *inDir == "" {
		flag.Usage()
		return fmt.Errorf("no input package provided")
	}

	err := compiler.Translate(*inDir, *outDir, *mainName)
	if err != nil {
		return err
	}

	return nil
}

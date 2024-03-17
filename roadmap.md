# Roadmap

## Compiler v1.0
* [x] Parser
* [x] Normalizer
* [x] Type solver
* [x] Bytecode compiler
* [x] Packages system
* [x] Check type constraints (`number`)
* [x] Check if all cases are exhaustive
* [x] Check if function parameters are exhaustive (for data types)
* [x] Smart import system
* [x] Support of \uNNNN string characters

## Compiler v...
* [ ] Nested record fields access
* [ ] Prefix operators (like infix ones, neg is ugly)
* [ ] "Tree shaking" to strip unused code
* [x] Compilation performance improvements
* [ ] Multithreaded compilation
  
## Libraries
* [x] Nar.Base library + tests
* [x] Nar.Program library
* [x] Nar.Time
* [x] Nar.Random
* [ ] Nar.Tests library
  * [x] Simple tests
  * [ ] Fuzz tests
* [ ] Unity plugin
* [ ] ...

## Quality Of Life
* [ ] Documentation
* [x] Language server
* [ ] Debugger
* [ ] Formatter
* IDE support
  * [x] Visual Studio Code
  * [ ] Jetbrains Family
  * [ ] Sublime
  * [ ] Vim

## Platforms
* JavaScript
  * [x] Runtime
  * [x] Linker
  * [ ] Replace with wasm from C
* Native
  * [x] CGo runtime
  * [x] Linker
  * [ ] Replace with C Runtime
* [ ] LLVM compiler

# Roadmap

## Compiler v0.1
* [x] Parser
* [x] Normalizer
* [x] Type solver
* [x] Bytecode compiler
* [x] Packages system

## Compiler v0.2
* [ ] Check type constraints (`num` and `cmp`)
* [ ] Check if all cases are exhaustive
* [ ] Check if function parameters are exhaustive (for data types)
* [ ] `export` definition keyword and tree shaking
* [ ] Nested record fields access
* [ ] Prefix operators (like infix ones)

## Libraries
* [x] Oak.Tests library
* [ ] Oak.Core library + tests
* [ ] Oak.Program library + tests
* [ ] Tests for compiler
* [ ] Oak.Leaf library (game engine)
* [ ] ...

## Quality Of Life
* [ ] Language server
* [ ] Debugger
* [ ] Formatter
* [ ] IDE support
  * [ ] Visual Studio Code
  * [ ] Jetbrains Family

## Platforms
* JavaScript
  * [x] Runtime
  * [x] Linker
* C99
  * [ ] Runtime (static library)
  * [ ] Linker
* [ ] LLVM compiler

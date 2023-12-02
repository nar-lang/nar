# Roadmap

## Compiler v0.1
* [x] Parser
* [x] Normalizer
* [x] Type solver
* [x] Bytecode compiler
* [x] Packages system

## Compiler v0.2
* [x] Check type constraints (`number`)
* [ ] Check if all cases are exhaustive
* [ ] Check if function parameters are exhaustive (for data types)
* [ ] `export` definition keyword and tree shaking
* [ ] Nested record fields access
* [ ] Prefix operators (like infix ones, neg is ugly)
* [ ] Do not require import if fully qualified name is used
* [ ] Support of \uNNNN string characters

## Libraries
* [x] Oak.Tests library
* [ ] Oak.Core library + tests
* [ ] Oak.Program library + tests
* [ ] Oak.Leaf library (game engine)
  * [ ] Oak.Leaf.GL
  * [ ] Oak.Leaf.Sprite
  * [ ] Oak.Leaf.Input
  * [ ] Oak.Leaf.UI
* [ ] ...

## Quality Of Life
* [ ] Documentation
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
  * [ ] Clean runtime from dependencies
* C99
  * [ ] Runtime (static library)
  * [ ] Linker
* [ ] LLVM compiler

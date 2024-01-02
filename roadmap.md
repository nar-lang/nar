# Roadmap

## Compiler v0.1
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

## Compiler v0.2
* [ ] Nested record fields access
* [ ] Prefix operators (like infix ones, neg is ugly)
* [ ] "tree shaking" to strip unused code
* [ ] Compilation performance improvements
  
## Libraries
* [x] Nar.Core library + tests
* [x] Nar.Program library
* [x] Nar.Time
* [ ] Nar.Random
* [ ] Nar.Tests library
  * [x] Simple tests
  * [ ] Fuzz tests
* [ ] Nar.Leaf library (game engine)
  * [ ] Nar.Leaf.GL
  * [ ] Nar.Leaf.Sprite
  * [ ] Nar.Leaf.Input
  * [ ] Nar.Leaf.UI
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

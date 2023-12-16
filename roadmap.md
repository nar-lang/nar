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

## Compiler v0.2
* [ ] Support of \uNNNN string characters
* [ ] Nested record fields access
* [ ] Prefix operators (like infix ones, neg is ugly)
* [ ] Imports improvements
  * [ ] Do not require import if fully qualified name is used
  * [ ] Do not require exposing import with name of the package, e.g. `import Oak.Core.String`~~`exposing (String)`~~`
  * [ ] Non-ambiguous identifiers can omit imports, e.g. `List.map`
  * [ ] Make prelude with default imports
* [ ] "tree shaking" to strip unused code
  
## Libraries
* [x] Oak.Core library + tests
* [ ] Oak.Program library
* [ ] Oak.Time
* [ ] Oak.Random
* [ ] Oak.Tests library
  * [x] Simple tests
  * [ ] Fuzz tests
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
* [ ] Compilation performance improvements
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

# Nar programming language

A typesafe functional scripting language.

Checkout the [Home page](https://nar-lang.com/) for more information.

## Compilation

1. Install [go compiler](https://go.dev) if you don't have it (version 1.21 or above is required).
2. Clone [nar-runtime-c](https://github.com/nar-lang/nar-runtime-c) and [nar](https://github.com/nar-lang/nar) repositories into the same directory: 
    ```bash
    git clone git@github.com:nar-lang/nar-runtime-c.git
    git clone git@github.com:nar-lang/nar.git
   ```
3. Compile nar-runtime-c using cmake:
    ```bash
    cd nar-runtime-c
    cmake . && make
    ```
4. Specify you home directory in nar/cmd/nar/nar.go:11 and :12. Note: this is temporary, until solution is found.
5. Compile nar using go compiler:
   ```bash
   cd ../nar
   CGO_ENABLED=1 go build -o ~/.nar/bin/nar ./cmd/nar/nar.go
   ```
   
## Installation

No additional installation required. Just put nar executable into your PATH if you want to use it globally.

## Help

If you got stuck, you can always ask for help in [Discussions](https://github.com/nar-lang/nar/discussions) or join
a [Discord server](https://discord.gg/sGNJVtNwpU).

If you got stuck, feel free to ask

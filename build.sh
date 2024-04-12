#!/bin/bash
cd "$(dirname "$0")"
CGO_ENABLED=1 go build -o ~/.nar/bin/nar ./cmd/nar/nar.go

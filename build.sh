#!/bin/bash


# Export the go binary
# export GOPATH=/Users/$USER/go
cd client/
cp $(go env GOROOT)/misc/wasm/wasm_exec.js .
GOOS=js GOARCH=wasm go build -o agent.wasm agent.go
cd ../

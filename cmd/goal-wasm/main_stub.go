//go:build !js || !wasm

// The goal-wasm entry point is built only for GOOS=js GOARCH=wasm (see
// main_wasm.go, which imports syscall/js). This stub gives the package a
// buildable file on every other platform so `go build ./...` and `go vet ./...`
// keep working on the host.
package main

func main() {}

//go:build js && wasm

// Command goal-wasm exposes the goal transpiler to JavaScript. Built for
// GOOS=js GOARCH=wasm, it installs a global `goalTranspile(src)` function that
// the playground calls on every Run — the whole front-end pipeline runs
// client-side in the browser (or in Node, for the CI gate), with no backend.
package main

import (
	"syscall/js"

	"goal/internal/backend"
)

// transpile is the JS-callable bridge. It takes one string argument (the .goal
// source) and returns an object { go, test, error }:
//
//   - go    — the transpiled Go (also carries the partial generated source when
//     lowering succeeded but the result failed to gofmt-parse).
//   - test  — the doctest sidecar (_test.go), empty when the source has none.
//   - error — a non-empty diagnostic string when transpilation failed.
func transpile(this js.Value, args []js.Value) any {
	res := map[string]any{"go": "", "test": "", "error": ""}
	if len(args) < 1 || args[0].Type() != js.TypeString {
		res["error"] = "goalTranspile: expected a single source string argument"
		return res
	}
	out, err := backend.Transpile(args[0].String())
	res["go"] = out.Go
	res["test"] = out.Test
	if err != nil {
		res["error"] = err.Error()
	}
	return res
}

func main() {
	js.Global().Set("goalTranspile", js.FuncOf(transpile))
	// Keep the module alive so the exported function stays callable for the
	// lifetime of the page.
	select {}
}

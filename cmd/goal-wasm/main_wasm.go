//go:build js && wasm

// Command goal-wasm exposes the goal transpiler to JavaScript. Built for
// GOOS=js GOARCH=wasm, it installs a global `goalTranspile(src)` function that
// the playground calls on every Run — the whole front-end pipeline runs
// client-side in the browser (or in Node, for the CI gate), with no backend.
package main

import (
	"fmt"
	"strings"
	"syscall/js"

	"goal/internal/backend"
	"goal/internal/parser"
	"goal/internal/sema"
)

// transpile is the JS-callable bridge. It takes the .goal source as its first
// argument and an optional source filename as its second (used to locate
// diagnostics, e.g. `traffic.goal:10:9: …`; defaults to source.goal). It returns
// an object { go, test, error }:
//
//   - go    — the transpiled Go (also carries the partial generated source when
//     lowering succeeded but the result failed to gofmt-parse).
//   - test  — the doctest sidecar (_test.go), empty when the source has none.
//   - error — a non-empty diagnostic string when the program is rejected.
//
// It mirrors `goal check`/goalc: the static checker runs first and an
// Error-severity diagnostic rejects the program before lowering, so the
// playground shows the same rejection a user gets from the CLI instead of
// silently transpiling invalid source.
func transpile(this js.Value, args []js.Value) any {
	res := map[string]any{"go": "", "test": "", "error": ""}
	if len(args) < 1 || args[0].Type() != js.TypeString {
		res["error"] = "goalTranspile: expected a source string argument"
		return res
	}
	source := args[0].String()
	sourceName := "source.goal"
	if len(args) >= 2 && args[1].Type() == js.TypeString && args[1].String() != "" {
		sourceName = args[1].String()
	}

	if diag, rejected := checkErrors(source, sourceName); rejected {
		res["error"] = diag
		return res
	}

	out, err := backend.Transpile(source)
	res["go"] = out.Go
	res["test"] = out.Test
	if err != nil {
		res["error"] = err.Error()
	}
	return res
}

// checkErrors runs the static checker and returns the rendered Error-severity
// diagnostics (one per line, located by sourceName) and whether the program is
// rejected. A parse failure renders in the shared `file:line:col: error:
// [syntax] …` form, matching goal check/build. Warnings never reject — they
// surface as backend warnings during lowering.
func checkErrors(source, sourceName string) (string, bool) {
	diags, err := sema.Analyze(source)
	if err != nil {
		pes := parser.CollectErrors(err)
		if len(pes) == 0 {
			return fmt.Sprintf("check: %v", err), true
		}
		lines := make([]string, 0, len(pes))
		for _, e := range pes {
			lines = append(lines, sema.Diagnostic{
				Pos:      e.Pos,
				Severity: sema.Severity(sema.Severity_Error{}),
				Code:     "syntax",
				Message:  e.Msg,
			}.Render(sourceName))
		}
		return strings.Join(lines, "\n"), true
	}
	var lines []string
	for _, d := range diags {
		if sema.HasErrors([]sema.Diagnostic{d}) {
			lines = append(lines, d.Render(sourceName))
		}
	}
	if len(lines) == 0 {
		return "", false
	}
	return strings.Join(lines, "\n"), true
}

func main() {
	js.Global().Set("goalTranspile", js.FuncOf(transpile))
	// Keep the module alive so the exported function stays callable for the
	// lifetime of the page.
	select {}
}

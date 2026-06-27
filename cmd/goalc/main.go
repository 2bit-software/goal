// Command goalc is the unified goal front-end: it reads a .goal program, runs the
// multi-feature lowering pipeline, and writes the transpiled Go.
package main

import (
	"fmt"
	"io"
	"os"

	"goal/internal/backend"
	"goal/internal/check"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "goalc:", err)
		os.Exit(1)
	}
}

// run transpiles the input named in args (or stdin when given "-") and writes the
// lowered Go to out. It first runs the static checker and writes any diagnostics to
// errOut; an Error-severity diagnostic rejects the program before lowering (unless
// -nocheck is given). With the -test flag it instead writes the doctest sidecar
// `_test.go`, erroring if the input has no doctests.
func run(args []string, stdin io.Reader, out, errOut io.Writer) error {
	testMode := false
	noCheck := false
	var files []string
	for _, a := range args {
		switch a {
		case "-test":
			testMode = true
		case "-nocheck":
			noCheck = true
		default:
			files = append(files, a)
		}
	}
	if len(files) != 1 {
		return fmt.Errorf("usage: goalc [-test] [-nocheck] <file.goal | ->")
	}
	src, err := readInput(files[0], stdin)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	if !noCheck {
		diags, err := check.Analyze(string(src))
		if err != nil {
			return fmt.Errorf("check: %w", err)
		}
		for _, d := range diags {
			fmt.Fprintln(errOut, d.Render(string(src), files[0]))
		}
		if check.HasErrors(diags) {
			return fmt.Errorf("%s rejected: %d checker error(s)", files[0], countErrors(diags))
		}
	}
	result, err := backend.Transpile(string(src))
	if err != nil {
		return err
	}
	if testMode {
		if result.Test == "" {
			return fmt.Errorf("no doctests in input")
		}
		_, err = fmt.Fprint(out, result.Test)
		return err
	}
	_, err = fmt.Fprint(out, result.Go)
	return err
}

// countErrors returns how many diagnostics are Error severity.
func countErrors(diags []check.Diagnostic) int {
	n := 0
	for _, d := range diags {
		if d.Severity == check.Error {
			n++
		}
	}
	return n
}

func readInput(name string, stdin io.Reader) ([]byte, error) {
	if name == "-" {
		return io.ReadAll(stdin)
	}
	return os.ReadFile(name)
}

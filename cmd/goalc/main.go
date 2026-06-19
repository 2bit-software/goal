// Command goalc is the unified goal front-end: it reads a .goal program, runs the
// multi-feature lowering pipeline, and writes the transpiled Go.
package main

import (
	"fmt"
	"io"
	"os"

	"goal/internal/pipeline"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "goalc:", err)
		os.Exit(1)
	}
}

// run transpiles the input named in args (or stdin when given "-") and writes the
// lowered Go to out. With the -test flag it instead writes the doctest sidecar
// `_test.go`, erroring if the input has no doctests.
func run(args []string, stdin io.Reader, out io.Writer) error {
	testMode := false
	var files []string
	for _, a := range args {
		if a == "-test" {
			testMode = true
			continue
		}
		files = append(files, a)
	}
	if len(files) != 1 {
		return fmt.Errorf("usage: goalc [-test] <file.goal | ->")
	}
	src, err := readInput(files[0], stdin)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	result, err := pipeline.Transpile(string(src))
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

func readInput(name string, stdin io.Reader) ([]byte, error) {
	if name == "-" {
		return io.ReadAll(stdin)
	}
	return os.ReadFile(name)
}

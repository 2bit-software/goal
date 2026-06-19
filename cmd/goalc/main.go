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

// run transpiles the input named in args (or stdin when given "-") and writes Go to
// out.
func run(args []string, stdin io.Reader, out io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: goalc <file.goal | ->")
	}
	src, err := readInput(args[0], stdin)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	goSrc, err := pipeline.Transpile(string(src))
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(out, goSrc)
	return err
}

func readInput(name string, stdin io.Reader) ([]byte, error) {
	if name == "-" {
		return io.ReadAll(stdin)
	}
	return os.ReadFile(name)
}

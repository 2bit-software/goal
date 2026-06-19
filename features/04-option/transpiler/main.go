package main

import (
	"fmt"
	"io"
	"os"
)

// main reads a .goal file named on the command line (or stdin when given "-")
// and writes the transpiled Go to stdout.
func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "transpile:", err)
		os.Exit(1)
	}
}

// run transpiles the input named in args and writes Go to out.
func run(args []string, stdin io.Reader, out io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: transpiler <file.goal | ->")
	}
	src, err := readInput(args[0], stdin)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	goSrc, err := transpile(string(src))
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

// Package pipeline defines the shared output shapes the goal front-end produces.
//
// The lowering itself lives in internal/backend (the AST engine): the legacy
// token-splice engine that once lived here was retired in US-043, leaving this
// package as the home of the engine-independent Output/GoFile/PackageOutput
// types and the source-position mapping (sourcemap.go).
package pipeline

// Output is the set of files the front-end produces from one source: the lowered
// Go, and an optional sibling `_test.go` extracted from doctests (empty when the
// source has none). Doctests are a side output — the driver supports N outputs,
// not one.
type Output struct {
	Go   string
	Test string
}

// GoFile is one generated Go source: the base file name to write and its formatted
// content. Names are derived from the originating `.goal` file (foo.goal -> foo.go,
// foo_test.go for its doctest sidecar); the synthesized prelude is goal_prelude.go.
type GoFile struct {
	Name string
	Go   string
}

// PackageOutput is the full Go output for one goal package, held in memory (no disk
// I/O): one Go file per source, the optional shared goal_prelude.go, and any doctest
// sidecars. The build driver decides whether to compile this from a temp dir or
// persist it via --emit.
type PackageOutput struct {
	Files []GoFile // transpiled sources, plus goal_prelude.go when the package uses closed-E Result
	Tests []GoFile // doctest sidecars (`_test.go`), one per source file that has doctests
}

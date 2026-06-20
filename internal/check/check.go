// Package check is the static checker: it proves the guarantees the front-end
// erases. Each goal feature lowers proven-valid input and defers (with a located
// error) anything it cannot resolve; this package is where each feature's static
// guarantee actually lands — exhaustiveness (02), field-completeness (08),
// `implements` satisfaction (07), closedness & From-totality (06), conversion
// totality (12), must-use (03), and the static-provable assert subset (10).
//
// # The spine is stable; the checks are slots
//
// This file is the contract — Diagnostic, Severity, Check, the Checks registry, and
// the Run driver — and changes rarely. Each guarantee lives in its own file
// (fields.go, exhaustive.go, …) behind a `func(src string, t *analyze.Tables)
// ([]Diagnostic, error)` already registered in Checks. Implementing a guarantee means
// filling in that one function and adding testdata; it does not mean touching this
// file or inventing the module shape. See CHECKER-TODO.md for the work queue and
// prompt.md for the per-check loop.
//
// # Design contract (decided once, do not re-litigate per check)
//
//   - No new parser. Checks reuse the existing lexer (internal/scan) and the
//     name-keyed symbol tables (internal/analyze), exactly like the lowering passes.
//     A check re-lexes the source and reads facts from analyze.Tables by name. Extend
//     the Tables when a check needs a fact they don't carry yet — never bolt a second
//     parser onto the project.
//   - Run on the ORIGINAL source, before any lowering. Lowering erases the very
//     structure a guarantee inspects (a `match` becomes a type switch; an incomplete
//     literal becomes expanded zero values), so checks must see the un-lowered text.
//     This mirrors how analyze.Build and doctest extraction already work.
//   - Positions are byte offsets into that original source. OffsetToPosition turns an
//     offset into a 1-based line/column for display; every Diagnostic carries the
//     offset so the driver and the CLI can render `file:line:col`.
//   - Defer, never guess. When a check cannot resolve a fact lexically (e.g. the type
//     of an untyped `x := match …`), emit a located Diagnostic saying so — a Warning
//     that names what could not be resolved — rather than assuming and risking a
//     false guarantee. A false "this is exhaustive" is worse than an honest "cannot
//     tell here." This is the same discipline the front-end already follows.
package check

import (
	"fmt"
	"sort"
	"strings"

	"goal/internal/analyze"
)

// Severity is how strongly a Diagnostic is enforced. An Error means the program is
// rejected (the guarantee is violated); a Warning is advisory (typically a deferral —
// something the checker could not resolve and is surfacing rather than silently
// assuming).
type Severity int

const (
	// Error marks a violated guarantee; the program is rejected.
	Error Severity = iota
	// Warning marks an advisory diagnostic, usually a located deferral.
	Warning
)

// String returns the lowercase severity label used in rendered diagnostics.
func (s Severity) String() string {
	if s == Warning {
		return "warning"
	}
	return "error"
}

// Diagnostic is one located finding. Pos is a byte offset into the original source;
// Feature names the guarantee that produced it (e.g. "02-match"); Code is a stable,
// greppable short identifier (e.g. "non-exhaustive-match"); Message is the
// agent/human-facing explanation.
type Diagnostic struct {
	Pos      int
	Severity Severity
	Feature  string
	Code     string
	Message  string
}

// Position is a 1-based line and column into the source.
type Position struct {
	Line int
	Col  int
}

// OffsetToPosition converts a byte offset into a 1-based line and column. An
// out-of-range offset is clamped to the nearest valid bound.
func OffsetToPosition(src string, off int) Position {
	if off < 0 {
		off = 0
	}
	if off > len(src) {
		off = len(src)
	}
	line := 1 + strings.Count(src[:off], "\n")
	col := off - (strings.LastIndexByte(src[:off], '\n') + 1) + 1
	return Position{Line: line, Col: col}
}

// Render formats a Diagnostic as `file:line:col: severity: [code] message`.
func (d Diagnostic) Render(src, filename string) string {
	p := OffsetToPosition(src, d.Pos)
	return fmt.Sprintf("%s:%d:%d: %s: [%s] %s", filename, p.Line, p.Col, d.Severity, d.Code, d.Message)
}

// Check is one named static check over the source. Its Run reads the same name-keyed
// tables the lowering passes use and returns located diagnostics (and an error only
// for an internal failure, not for a rejected program — a violated guarantee is a
// Diagnostic, not an error).
type Check struct {
	Name string
	Run  func(src string, t *analyze.Tables) ([]Diagnostic, error)
}

// Checks is the ordered registry of every static check. Each entry points at a slot
// function in this package; an unimplemented slot returns no diagnostics, so the
// checker is always safe to run. Order is roughly by self-containment / value (see
// CHECKER-TODO.md): the most local, type-inference-free guarantees first.
var Checks = []Check{
	{Name: "fields", Run: checkFields},         // 08-no-zero-value: field-completeness
	{Name: "exhaustive", Run: checkExhaustive}, // 02-match: match exhaustiveness
	{Name: "implements", Run: checkImplements}, // 07-implements: interface satisfaction
	{Name: "closed", Run: checkClosed},         // 06-error-e: closedness & From-totality
	{Name: "convert", Run: checkConvert},       // 12-derive-convert: conversion totality
	{Name: "mustuse", Run: checkMustUse},       // 03-result: must-use
	{Name: "assert", Run: checkAssert},         // 10-assert: static-provable subset
}

// Run executes every registered check against pre-built tables and returns the
// accumulated diagnostics, sorted by source position. It fails only if a check
// reports an internal error; a rejected program surfaces as Error-severity
// diagnostics, not as a returned error.
func Run(src string, t *analyze.Tables) ([]Diagnostic, error) {
	var diags []Diagnostic
	for _, c := range Checks {
		ds, err := c.Run(src, t)
		if err != nil {
			return nil, fmt.Errorf("check %s: %w", c.Name, err)
		}
		diags = append(diags, ds...)
	}
	sort.SliceStable(diags, func(i, j int) bool { return diags[i].Pos < diags[j].Pos })
	return diags, nil
}

// Analyze builds the tables from src and runs every check — the convenience entry
// point for callers that don't already hold an analyze.Tables.
func Analyze(src string) ([]Diagnostic, error) {
	return Run(src, analyze.Build(src))
}

// HasErrors reports whether any diagnostic is Error severity.
func HasErrors(diags []Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == Error {
			return true
		}
	}
	return false
}

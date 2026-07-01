// Package backendtest holds Go-authored behavioral tests for the goal Go backend
// that the golden corpus cannot express — chiefly the inline `...derive` spread's
// transpile-error cases (the corpus compares successful goldens only) and a parity
// check that the inline form emits the same conversion statements as a `derive func`.
//
// It lives in its own package (not internal/backend) because internal/backend is
// generated from emit.goal and guarded by the verify-generated drift gate, which a
// hand-written _test.go beside the generated files would trip.
package backendtest

// Package dep is a US-021 fixture dependency: a plain Go package exposing an
// exported single-value function (One) and an exported error-returning function
// (Get). EnrichForeign reads these from the .go source into info.FuncSignatures
// keyed `dep.One` / `dep.Get`, so question checking can evaluate `?` applied to a
// cross-package call against the real foreign arity.
package dep

// One returns a single value with no trailing error. `?` on `dep.One()` inside a
// Result[_, error] function is misuse (nothing to propagate) and must be flagged.
func One() int { return 1 }

// Get returns (T, error): a well-formed `?` callee. `?` on `dep.Get()` is clean.
func Get() (int, error) { return 1, nil }

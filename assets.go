// Package goal exposes embedded repository documents to the toolchain.
//
// The AI guide (goal ai) is assembled in-process from these documents and the live
// transpiler, so it ships inside the binary and works from an installed goal anywhere
// — no files are read from disk at runtime. Embedding lives at the module root because
// an embed directive cannot reach above its own package directory, and the documents
// live under docs/ at the root.
package goal

import "embed"

// Docs holds the documents the AI guide renders: the by-example feature source, the
// overview, and the authored guide fragments under docs/ai.
//
//go:embed docs/by-example.md docs/overview.md docs/ai
var Docs embed.FS

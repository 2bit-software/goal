#!/usr/bin/env bash
# Build the goal playground into site/: compile the transpiler to WebAssembly,
# copy Go's matching wasm_exec.js, and regenerate features.json from
# docs/by-example.md (which also verifies every example against the live
# transpiler). The three generated artifacts (goal.wasm, wasm_exec.js,
# features.json) are gitignored — rebuild with this script.
set -euo pipefail

cd "$(dirname "$0")/.."

echo "==> building goal.wasm"
GOOS=js GOARCH=wasm go build -o site/goal.wasm ./cmd/goal-wasm

echo "==> copying wasm_exec.js from $(go env GOROOT)"
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" site/wasm_exec.js

echo "==> generating site/features.json from docs/by-example.md"
go run ./cmd/build-playground -in docs/by-example.md -out site/features.json

echo "==> done. Serve with:  (cd site && python3 -m http.server 8000)"

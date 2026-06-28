# Verification — US-017

## verifyCommands (from prd.json)
- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test ./... -count=1` — PASS (all packages, incl. new `goal/internal/parser`)

## Acceptance Criteria
1. internal/parser parses package clause, imports, and func/type/var/const
   declarations of the Go subset into ast.File — DONE (`ParseFile`).
2. A test parses a sample file and asserts the declaration list shape — DONE
   (`TestParseFileDeclarationShape` asserts package name, 5 imports incl.
   named/blank/dot, 12 ordered decls with GenDecl tokens + spec names, and the
   plain func + method shapes; plus struct/alias/interface and composite-initializer
   tests, and error-path tests).

No CRITICAL/MAJOR findings.

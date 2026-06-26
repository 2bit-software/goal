package check

import "testing"

// TestBuiltinSumMatchNotDeferred guards the §2 exhaustiveness check against a false
// positive: a `match` on the builtin sum types `Result`/`Option` must not be reported as
// an unresolved enum. Their arms (`Result.Ok`, `Option.Some`) are qualified, so they
// reach the enum-resolution path, but `Result`/`Option` are never declared as user enums —
// their exhaustiveness is owned by the result/option lowering passes (a missing Ok/Err or
// Some/None arm is a pass error). (Regression: a complete `match` on `Result` reported
// `unresolved-match-enum`.)
func TestBuiltinSumMatchNotDeferred(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"result", `package x

func decide() Result[int, error] { return Result.Ok(0) }

func run() int {
	match decide() {
		Result.Ok(v) => { return v }
		Result.Err(e) => { return 1 }
	}
}
`},
		{"option", `package x

func lookup() Option[int] { return Option.None }

func run() int {
	match lookup() {
		Option.Some(v) => { return v }
		Option.None => { return 0 }
	}
}
`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			diags, err := Analyze(c.src)
			if err != nil {
				t.Fatalf("Analyze: %v", err)
			}
			if hasCode(diags, "unresolved-match-enum") {
				for _, d := range diags {
					t.Logf("diag: [%s] %s", d.Code, d.Message)
				}
				t.Errorf("a match on the builtin sum type was deferred as an unresolved enum")
			}
		})
	}
}

// TestUserEnumStillDeferred confirms the builtin skip did not blind the check to a genuine
// out-of-file user enum: a match whose arms name an enum not declared in this file must
// still defer with a located warning.
func TestUserEnumStillDeferred(t *testing.T) {
	const src = `package x

func run(s Status) int {
	match s {
		Status.Pending => { return 0 }
		Status.Active(a) => { return 1 }
	}
}
`
	diags, err := Analyze(src)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if !hasCode(diags, "unresolved-match-enum") {
		t.Errorf("a match on an undeclared user enum should still defer with unresolved-match-enum")
	}
}

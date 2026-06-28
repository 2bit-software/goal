package corpus

import (
	"goal/internal/sema"
)

// SemaCheck is the AST-based checker the corpus judges against the inline // want
// markers. It parses src with the front-end, resolves its semantic facts, and runs
// every sema check, returning the located sema diagnostics directly. It is the sole
// checker the corpus drives now that the legacy lexical checker (internal/check) is
// deleted; as later sema stories extend sema.Check, this entry point carries their
// diagnostics too. SemaCheck satisfies the checker shape RunCheck expects.
func SemaCheck(src string) ([]sema.Diagnostic, error) {
	return sema.Analyze(src)
}

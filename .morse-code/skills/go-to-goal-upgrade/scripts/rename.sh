#!/usr/bin/env bash
# rename.sh — Step 1 of the go->goal upgrade: rename .go -> .goal within scope.
#
# Usage: rename.sh <target>
#   <target> is one .go file OR a single package directory (run scope-guard.sh first).
#
# What it does:
#   * renames each <name>.go -> <name>.goal (git mv when in a repo, else mv)
#   * leaves the `package <name>` clause unchanged (goal keeps Go's package clause)
#   * preserves build-tag comments verbatim (`//go:build` / `// +build` are
#     comments and pass through; any found are reported for manual review)
#   * scans for reserved-word identifier collisions and REPORTS them — goal
#     reserves `match`, `enum`, `assert` beyond Go's keywords; Go source using any
#     of those as a BARE identifier is rejected by the goal parser and must be
#     renamed (e.g. `enum` -> `enumDecl`). `enumOf`/`enumName`/`.Enum` are fine.
#
# It renames and reports; it does NOT auto-rewrite reserved identifiers (that is a
# semantic edit a human/agent should make), and does NOT run goal fix (Step 2).
set -euo pipefail

if [ $# -ne 1 ]; then
  echo "usage: rename.sh <target-file-or-dir>" >&2
  exit 1
fi
target="$1"

in_git() { git rev-parse --is-inside-work-tree >/dev/null 2>&1; }
do_mv() {
  if in_git; then git mv "$1" "$2" 2>/dev/null || mv "$1" "$2"; else mv "$1" "$2"; fi
}

# Collect the .go files in scope (exclude tests by default; tests usually live
# with the package and can be renamed too, but flag them separately).
files=()
if [ -f "$target" ]; then
  files=("$target")
elif [ -d "$target" ]; then
  while IFS= read -r line; do
    [ -n "$line" ] && files+=("$line")
  done < <(find "$target" -maxdepth 1 -type f -name '*.go' | sort)
else
  echo "error: no such file or directory: $target" >&2
  exit 1
fi

if [ "${#files[@]}" -eq 0 ]; then
  echo "error: no .go files to rename in $target" >&2
  exit 1
fi

echo "== rename report =="
RESERVED='\b(match|enum|assert)\b'
for f in "${files[@]}"; do
  goalf="${f%.go}.goal"

  # Build-tag report (comments pass through, but call them out).
  if grep -qE '^//(go:build| \+build)' "$f"; then
    echo "  [build-tag] $f carries build-tag comments (preserved verbatim; review for goal support)"
  fi

  # Reserved-word collision report: bare-word uses (not enumOf/enumName/.Enum).
  # Heuristic: the reserved word as a standalone token, not preceded by '.' and
  # not immediately followed by an identifier char. Comments are stripped first
  # (everything from `//` to EOL) so prose mentioning enum/match is not flagged;
  # this is a heuristic — review the hits, it does not parse Go.
  hits=$(sed 's://.*$::' "$f" | grep -nE "(^|[^.A-Za-z0-9_])(match|enum|assert)([^A-Za-z0-9_]|$)" \
      | grep -vE 'enum(Of|Name|Decl)|\.Enum' || true)
  if [ -n "$hits" ]; then
    echo "  [reserved] $f uses a reserved word (match/enum/assert) as a possible bare identifier — review and rename before parsing:"
    printf '%s\n' "$hits" | sed 's/^/      /'
  fi

  do_mv "$f" "$goalf"
  echo "  renamed $f -> $goalf"
done
echo "== done; package clause(s) unchanged; run 'goal check <scope>' then 'goal fix' (Step 2) =="

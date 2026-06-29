#!/usr/bin/env bash
# scope-guard.sh — classify an upgrade target and refuse a multi-package module.
#
# Usage: scope-guard.sh <target>
#   <target> is one .go file OR a directory.
#
# Prints exactly one of:
#   FILE <path>                 — a single .go file (in scope)
#   PACKAGE <dir> <pkgname>     — a single-package directory (in scope)
#   MODULE <reason>             — a multi-package module (REFUSED) and exits 2
# Exits 0 for FILE/PACKAGE, 2 for MODULE, 1 on usage/IO error.
#
# The supported unit is ONE .go file or ONE go.mod package directory — the same
# per-package model the goal self-host idiomatization used. Pointing the upgrade
# at a whole multi-package module is refused: run it per package instead.
set -euo pipefail

if [ $# -ne 1 ]; then
  echo "usage: scope-guard.sh <target-file-or-dir>" >&2
  exit 1
fi
target="$1"

if [ ! -e "$target" ]; then
  echo "error: no such file or directory: $target" >&2
  exit 1
fi

# Single file case.
if [ -f "$target" ]; then
  case "$target" in
    *.go) echo "FILE $target"; exit 0 ;;
    *) echo "error: not a .go file: $target" >&2; exit 1 ;;
  esac
fi

# Directory case: count distinct package directories that contain .go files.
# A single package = .go files in exactly ONE directory within the target
# (a subdirectory with its own .go files is a second package = a module).
dirs_with_go=$(find "$target" -type f -name '*.go' \
  ! -name '*_test.go' -exec dirname {} \; | sort -u)
n=$(printf '%s\n' "$dirs_with_go" | grep -c . || true)

if [ "$n" -eq 0 ]; then
  echo "error: no .go files found under $target" >&2
  exit 1
fi

if [ "$n" -gt 1 ]; then
  echo "MODULE multi-package module: found .go files in $n directories under $target; upgrade one package at a time (the supported unit is a single .go file or a single package directory)"
  exit 2
fi

# Exactly one package directory. Verify it's a single Go package (one package name).
pkgnames=$(grep -hE '^package [A-Za-z_][A-Za-z0-9_]*' "$target"/*.go 2>/dev/null \
  | awk '{print $2}' | sort -u)
np=$(printf '%s\n' "$pkgnames" | grep -c . || true)
if [ "$np" -gt 1 ]; then
  echo "MODULE directory $target contains multiple package names ($(echo "$pkgnames" | tr '\n' ' ')); not a single package"
  exit 2
fi

echo "PACKAGE $target ${pkgnames:-?}"
exit 0

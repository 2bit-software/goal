package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const plainGoFix = `package app

func load(p string) ([]byte, error) {
	f, err := read(p)
	if err != nil {
		return nil, err
	}
	return f, nil
}
`

// `goal fix` on a single file prints the idiomatic rewrite to stdout and writes nothing.
func TestFixStdoutLeavesFileUnchanged(t *testing.T) {
	dir := goalModule(t, map[string]string{"app/load.goal": plainGoFix})
	file := filepath.Join(dir, "app", "load.goal")

	var out, errOut bytes.Buffer
	if err := run([]string{"fix", file}, &out, &errOut); err != nil {
		t.Fatalf("fix failed: %v\n%s", err, errOut.String())
	}
	if !strings.Contains(out.String(), "Result[[]byte, error]") || !strings.Contains(out.String(), "read(p)?") {
		t.Fatalf("stdout did not contain the idiomatic rewrite:\n%s", out.String())
	}
	on, _ := os.ReadFile(file)
	if string(on) != plainGoFix {
		t.Fatalf("default mode must not modify the file on disk, but it changed:\n%s", on)
	}
}

// `goal fix -inplace` writes the rewrite back and reports the file as fixed.
func TestFixInplaceWritesFile(t *testing.T) {
	dir := goalModule(t, map[string]string{"app/load.goal": plainGoFix})
	file := filepath.Join(dir, "app", "load.goal")

	var out, errOut bytes.Buffer
	if err := run([]string{"fix", "-inplace", file}, &out, &errOut); err != nil {
		t.Fatalf("fix -inplace failed: %v\n%s", err, errOut.String())
	}
	on, _ := os.ReadFile(file)
	if !strings.Contains(string(on), "Result[[]byte, error]") {
		t.Fatalf("-inplace did not rewrite the file:\n%s", on)
	}
	if !strings.Contains(out.String(), "fixed") {
		t.Fatalf("expected a 'fixed' line on stdout, got:\n%s", out.String())
	}
}

// `goal fix -inplace` on an already-idiomatic file makes no change and reports nothing.
func TestFixInplaceNoChange(t *testing.T) {
	const idiomatic = `package app

func load(p string) Result[[]byte, error] {
	f := read(p)?
	return Result.Ok(f)
}
`
	dir := goalModule(t, map[string]string{"app/load.goal": idiomatic})
	file := filepath.Join(dir, "app", "load.goal")
	info, _ := os.Stat(file)
	before := info.ModTime()

	var out, errOut bytes.Buffer
	if err := run([]string{"fix", "-inplace", file}, &out, &errOut); err != nil {
		t.Fatalf("fix failed: %v\n%s", err, errOut.String())
	}
	if strings.Contains(out.String(), "fixed") {
		t.Fatalf("unchanged file should not be reported as fixed:\n%s", out.String())
	}
	info2, _ := os.Stat(file)
	if !info2.ModTime().Equal(before) {
		t.Fatal("unchanged file was rewritten (mtime changed)")
	}
}

// Changing an exported function's signature emits a warning to stderr.
func TestFixExportedWarning(t *testing.T) {
	const exported = `package app

func Load(p string) ([]byte, error) {
	f, err := read(p)
	if err != nil {
		return nil, err
	}
	return f, nil
}
`
	dir := goalModule(t, map[string]string{"app/load.goal": exported})
	var out, errOut bytes.Buffer
	if err := run([]string{"fix", filepath.Join(dir, "app", "load.goal")}, &out, &errOut); err != nil {
		t.Fatalf("fix failed: %v\n%s", err, errOut.String())
	}
	if !strings.Contains(errOut.String(), "exported") || !strings.Contains(errOut.String(), "Load") {
		t.Fatalf("expected an exported-signature warning on stderr, got:\n%s", errOut.String())
	}
}

// A non-existent path is an operational error.
func TestFixBadPath(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := run([]string{"fix", "/no/such/path.goal"}, &out, &errOut); err == nil {
		t.Fatal("expected an error for a missing path")
	}
}

// `goal fix` discovers and rewrites every .goal file under a directory.
func TestFixDirectory(t *testing.T) {
	dir := goalModule(t, map[string]string{"app/load.goal": plainGoFix})
	var out, errOut bytes.Buffer
	if err := run([]string{"fix", dir}, &out, &errOut); err != nil {
		t.Fatalf("fix dir failed: %v\n%s", err, errOut.String())
	}
	if !strings.Contains(out.String(), "Result[[]byte, error]") {
		t.Fatalf("directory fix did not rewrite the package:\n%s", out.String())
	}
}

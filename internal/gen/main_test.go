package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestGeneratedFilesUpToDate re-renders both generated files into a temp
// directory and diffs them against the committed zz_gen_generic-api.go
// and zz_gen_generic-api_test.go, so a stale commit (one not produced by
// the current templates/arity.go) fails CI instead of silently drifting.
func TestGeneratedFilesUpToDate(t *testing.T) {
	dir := t.TempDir()
	mainTmp := filepath.Join(dir, mainOutName)
	testTmp := filepath.Join(dir, testOutName)

	if err := run(mainTmp, testTmp); err != nil {
		t.Fatalf("run: %v", err)
	}

	checkMatches(t, mainTmp, filepath.Join("..", "..", mainOutName))
	checkMatches(t, testTmp, filepath.Join("..", "..", testOutName))
}

// checkMatches fails the test if got (freshly generated) and want
// (committed) differ.
func checkMatches(t *testing.T, gotPath, wantPath string) {
	t.Helper()

	got, err := os.ReadFile(gotPath)
	if err != nil {
		t.Fatalf("read generated %s: %v", gotPath, err)
	}
	want, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("read committed %s: %v", wantPath, err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s is stale; run `go generate ./...`", wantPath)
	}
}

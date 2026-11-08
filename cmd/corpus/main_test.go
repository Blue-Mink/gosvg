package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeSVG(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCollectSkipsAppleDoubleAndNonSVG(t *testing.T) {
	dir := t.TempDir()
	writeSVG(t, dir, "a.svg", `<svg viewBox="0 0 4 4"><rect width="4" height="4"/></svg>`)
	writeSVG(t, dir, "._a.svg", "binary junk")
	writeSVG(t, dir, "note.txt", "x")
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	writeSVG(t, sub, "b.svg", `<svg viewBox="0 0 4 4"><rect width="4" height="4"/></svg>`)

	files, err := collect([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("collect = %d files (%v), want 2", len(files), files)
	}
}

func TestAnalyzeCounts(t *testing.T) {
	dir := t.TempDir()
	writeSVG(t, dir, "ok.svg", `<svg viewBox="0 0 4 4"><rect width="4" height="4" fill="red"/></svg>`)
	writeSVG(t, dir, "empty.svg", `<svg viewBox="0 0 4 4"/>`)
	writeSVG(t, dir, "bad.svg", `not xml at all`)

	rep := analyze(mustCollect(t, dir), 16)
	if rep.Total != 3 {
		t.Fatalf("Total = %d, want 3", rep.Total)
	}
	if rep.WithContent != 1 {
		t.Fatalf("WithContent = %d, want 1", rep.WithContent)
	}
	if rep.Blank != 1 {
		t.Fatalf("Blank = %d, want 1", rep.Blank)
	}
	if rep.ParseFailed != 1 {
		t.Fatalf("ParseFailed = %d, want 1", rep.ParseFailed)
	}
	if len(rep.FailedFiles) != 1 {
		t.Fatalf("FailedFiles = %v, want 1 entry", rep.FailedFiles)
	}
}

func TestAnalyzeRecordsDiagnostics(t *testing.T) {
	dir := t.TempDir()
	writeSVG(t, dir, "f.svg", `<svg viewBox="0 0 4 4"><filter id="x"/><rect width="4" height="4" fill="red"/></svg>`)
	rep := analyze(mustCollect(t, dir), 8)
	if rep.Diagnostics["unsupported element skipped: filter"] != 1 {
		t.Fatalf("diagnostics = %v", rep.Diagnostics)
	}
}

func TestNormalizeDiag(t *testing.T) {
	if got := normalizeDiag("mask reference not found: abc"); got != "mask reference not found: <id>" {
		t.Fatalf("normalizeDiag = %q", got)
	}
	if got := normalizeDiag("unsupported element skipped: pattern"); got != "unsupported element skipped: pattern" {
		t.Fatalf("element name should be kept, got %q", got)
	}
}

func mustCollect(t *testing.T, dir string) []string {
	t.Helper()
	files, err := collect([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

package main

import (
	"os"
	"path/filepath"
	"testing"
)

const goodSVG = `<svg viewBox="0 0 4 4"><rect width="4" height="4" fill="red"/></svg>`

func TestParseArgs(t *testing.T) {
	for _, tc := range []struct {
		args    []string
		in, out string
	}{
		{[]string{"cmd"}, "testdata", ""},
		{[]string{"cmd", "in"}, "in", ""},
		{[]string{"cmd", "in", "out"}, "in", "out"},
	} {
		in, out := parseArgs(tc.args)
		if in != tc.in || out != tc.out {
			t.Errorf("parseArgs(%v) = (%q,%q), want (%q,%q)", tc.args, in, out, tc.in, tc.out)
		}
	}
}

func TestRunRendersSVGsOnly(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.svg"), []byte(goodSVG), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skip.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	ok, failed, err := run(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if ok != 1 || failed != 0 {
		t.Fatalf("ok=%d failed=%d, want 1/0", ok, failed)
	}
}

func TestRunWritesPNG(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.svg"), []byte(goodSVG), 0o644); err != nil {
		t.Fatal(err)
	}
	ok, failed, err := run(dir, out)
	if err != nil {
		t.Fatal(err)
	}
	if ok != 1 || failed != 0 {
		t.Fatalf("ok=%d failed=%d, want 1/0", ok, failed)
	}
	st, err := os.Stat(filepath.Join(out, "a.png"))
	if err != nil {
		t.Fatalf("png not written: %v", err)
	}
	if st.Size() == 0 {
		t.Fatal("png is empty")
	}
}

func TestRunMissingDir(t *testing.T) {
	if _, _, err := run(filepath.Join(t.TempDir(), "nope"), ""); err == nil {
		t.Fatal("expected error for missing input dir")
	}
}

func TestRunEmptyDir(t *testing.T) {
	ok, failed, err := run(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	if ok != 0 || failed != 0 {
		t.Fatalf("empty dir: ok=%d failed=%d, want 0/0", ok, failed)
	}
}

func TestRunCountsRenderFailure(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bad.svg"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	ok, failed, err := run(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if ok != 0 || failed != 1 {
		t.Fatalf("ok=%d failed=%d, want 0/1", ok, failed)
	}
}

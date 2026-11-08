package gosvg

import (
	"strings"
	"testing"
)

func TestParseASCIIEncodingDeclaration(t *testing.T) {
	svg := `<?xml version="1.0" encoding="ASCII"?><svg viewBox="0 0 10 10"><rect width="10" height="10" fill="red"/></svg>`
	doc, err := Parse([]byte(svg))
	if err != nil {
		t.Fatalf("ASCII declaration should parse: %v", err)
	}
	img, err := doc.Render(8, 8)
	if err != nil {
		t.Fatal(err)
	}
	if img.NRGBAAt(4, 4).A == 0 {
		t.Fatal("expected rendered content")
	}
}

func TestParseUTF8DeclarationStillWorks(t *testing.T) {
	svg := `<?xml version="1.0" encoding="UTF-8"?><svg viewBox="0 0 10 10"><rect width="10" height="10" fill="red"/></svg>`
	if _, err := Parse([]byte(svg)); err != nil {
		t.Fatalf("UTF-8 declaration: %v", err)
	}
}

func TestParseLatin1Declaration(t *testing.T) {
	raw := []byte(`<?xml version="1.0" encoding="ISO-8859-1"?><svg viewBox="0 0 24 10"><text x="1" y="8" font-size="8">caf`)
	raw = append(raw, 0xE9) // Latin-1 的 é
	raw = append(raw, []byte(`</text></svg>`)...)
	doc, err := Parse(raw)
	if err != nil {
		t.Fatalf("Latin-1 should parse: %v", err)
	}
	if _, err := doc.Render(48, 20); err != nil {
		t.Fatal(err)
	}
}

func TestParseUnsupportedCharsetNamesIt(t *testing.T) {
	svg := `<?xml version="1.0" encoding="SHIFT_JIS"?><svg viewBox="0 0 10 10"><rect width="10" height="10"/></svg>`
	_, err := Parse([]byte(svg))
	if err == nil {
		t.Fatal("expected error for unsupported charset")
	}
	if !strings.Contains(err.Error(), "SHIFT_JIS") {
		t.Fatalf("error should name the charset, got: %v", err)
	}
}

func TestCharsetReaderCanBeDisabled(t *testing.T) {
	old := CharsetReader
	CharsetReader = nil
	defer func() { CharsetReader = old }()
	svg := `<?xml version="1.0" encoding="ASCII"?><svg viewBox="0 0 10 10"><rect width="10" height="10"/></svg>`
	if _, err := Parse([]byte(svg)); err == nil {
		t.Fatal("with CharsetReader=nil, non-UTF-8 declaration should fail (encoding/xml default)")
	}
}

func TestDefaultCharsetReaderUnknown(t *testing.T) {
	if _, err := defaultCharsetReader("KOI8-R", strings.NewReader("x")); err == nil {
		t.Fatal("expected error for unknown charset")
	}
}

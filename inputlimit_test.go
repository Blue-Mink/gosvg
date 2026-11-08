package gosvg

import (
	"errors"
	"strings"
	"testing"
)

func TestParseInputTooLarge(t *testing.T) {
	big := make([]byte, MaxSVGSize+1)
	copy(big, `<svg viewBox="0 0 10 10"></svg>`)
	_, err := Parse(big)
	if !errors.Is(err, ErrInputTooLarge) {
		t.Fatalf("err = %v, want ErrInputTooLarge", err)
	}
	var ue *UnsupportedError
	if errors.As(err, &ue) && ue.Code != CodeInputTooLarge {
		t.Fatalf("Code = %v, want CodeInputTooLarge", ue.Code)
	}
}

func TestParseTooManyElements(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`<svg viewBox="0 0 10 10">`)
	for i := 0; i <= MaxElementCount; i++ {
		sb.WriteString(`<rect width="1" height="1"/>`)
	}
	sb.WriteString(`</svg>`)
	data := []byte(sb.String())
	if len(data) > MaxSVGSize {
		t.Fatalf("test data %d bytes exceeds MaxSVGSize; shrink per-element text", len(data))
	}
	if _, err := Parse(data); !errors.Is(err, ErrTooManyElements) {
		t.Fatalf("err = %v, want ErrTooManyElements", err)
	}
}

func TestParseNestingTooDeep(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`<svg viewBox="0 0 10 10">`)
	for i := 0; i <= MaxXMLDepth; i++ {
		sb.WriteString(`<g>`)
	}
	sb.WriteString(`<rect width="1" height="1"/>`)
	for i := 0; i <= MaxXMLDepth; i++ {
		sb.WriteString(`</g>`)
	}
	sb.WriteString(`</svg>`)
	if _, err := Parse([]byte(sb.String())); !errors.Is(err, ErrNestingTooDeep) {
		t.Fatalf("err = %v, want ErrNestingTooDeep", err)
	}
}

func TestParseWithinLimitsStillWorks(t *testing.T) {
	svg := `<svg viewBox="0 0 10 10"><g><rect width="1" height="1" fill="red"/></g></svg>`
	if _, err := Parse([]byte(svg)); err != nil {
		t.Fatalf("normal input should parse: %v", err)
	}
}

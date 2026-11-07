package gosvg

import "testing"

func TestRegisteredFontsSorted(t *testing.T) {
	fonts := RegisteredFonts()
	for i := 1; i < len(fonts); i++ {
		if fonts[i] < fonts[i-1] {
			t.Fatalf("RegisteredFonts not sorted: %v", fonts)
		}
	}
}

func TestResetFonts(t *testing.T) {
	ResetFonts()
	if got := RegisteredFonts(); len(got) != 0 {
		t.Fatalf("after ResetFonts: %v, want empty", got)
	}
	ResetFonts() // 幂等
	if got := RegisteredFonts(); len(got) != 0 {
		t.Fatalf("after second ResetFonts: %v, want empty", got)
	}
}

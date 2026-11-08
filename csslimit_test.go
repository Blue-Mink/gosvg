package gosvg

import (
	"strconv"
	"strings"
	"testing"
)

func TestParseCSSRuleLimit(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < MaxCSSRules+5; i++ {
		sb.WriteString(".c" + strconv.Itoa(i) + "{fill:red}")
	}
	rules, truncated := parseCSSLimited(sb.String())
	if !truncated {
		t.Fatal("expected truncated=true")
	}
	if len(rules) != MaxCSSRules {
		t.Fatalf("rules = %d, want %d", len(rules), MaxCSSRules)
	}
	if n := len(parseCSS(sb.String())); n != MaxCSSRules {
		t.Fatalf("parseCSS wrapper returned %d rules, want %d", n, MaxCSSRules)
	}
}

func TestParseCSSSelectorLenLimit(t *testing.T) {
	long := ".c" + strings.Repeat("x", MaxCSSSelectorLen+10)
	rules, truncated := parseCSSLimited(long + "{fill:red}")
	if !truncated {
		t.Fatal("expected truncated=true for over-long selector")
	}
	if len(rules) != 0 {
		t.Fatalf("rules = %d, want 0", len(rules))
	}
}

func TestParseCSSDeclLenLimit(t *testing.T) {
	decl := "fill:red;" + strings.Repeat("a:1;", MaxCSSDeclLen/4+10)
	rules, truncated := parseCSSLimited(".c{" + decl + "}")
	if !truncated {
		t.Fatal("expected truncated=true for over-long declaration block")
	}
	if len(rules) != 0 {
		t.Fatalf("rules = %d, want 0", len(rules))
	}
}

func TestCSSLimitProducesDiagnostic(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`<svg viewBox="0 0 4 4"><style>`)
	for i := 0; i < MaxCSSRules+1; i++ {
		sb.WriteString(".c" + strconv.Itoa(i) + "{fill:red}")
	}
	sb.WriteString(`</style><rect width="4" height="4" fill="blue"/></svg>`)
	doc, err := Parse([]byte(sb.String()))
	if err != nil {
		t.Fatal(err)
	}
	if joined := strings.Join(doc.Diagnostics(), "\n"); !strings.Contains(joined, "exceeded size limits") {
		t.Fatalf("missing CSS limit diagnostic; got:\n%s", joined)
	}
	// 渲染仍应正常工作（截断是降级，不是失败）。
	if _, err := doc.Render(8, 8); err != nil {
		t.Fatalf("render: %v", err)
	}
}

func TestParseCSSNormalUnchanged(t *testing.T) {
	rules, truncated := parseCSSLimited(".a{fill:red} rect.b{stroke:blue}")
	if truncated {
		t.Fatal("normal CSS should not be truncated")
	}
	if len(rules) != 2 {
		t.Fatalf("rules = %d, want 2", len(rules))
	}
}

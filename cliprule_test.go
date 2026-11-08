package gosvg

import "testing"

// clipPath 内单个 path 的两个重叠子路径：evenodd 下重叠区（winding=2）被排除，
// nonzero 下全部保留。真实语料里 clip-rule="evenodd" 出现 128 次，此前完全被忽略。
const clipRuleSVG = `<svg viewBox="0 0 10 10">
  <defs><clipPath id="c">%s</clipPath></defs>
  <rect width="10" height="10" fill="red" clip-path="url(#c)"/>
</svg>`

func renderClip(t *testing.T, inner string) (center, corner uint8) {
	t.Helper()
	img, err := Render([]byte(sprintfClip(inner)), 20)
	if err != nil {
		t.Fatal(err)
	}
	return img.NRGBAAt(10, 10).A, img.NRGBAAt(2, 2).A
}

func sprintfClip(inner string) string {
	return `<svg viewBox="0 0 10 10">
  <defs><clipPath id="c">` + inner + `</clipPath></defs>
  <rect width="10" height="10" fill="red" clip-path="url(#c)"/>
</svg>`
}

func TestClipRuleEvenOddOnPath(t *testing.T) {
	c, corner := renderClip(t, `<path clip-rule="evenodd" d="M0 0h10v10H0z M3 3h4v4H3z"/>`)
	if corner == 0 {
		t.Fatal("outer region should be kept")
	}
	if c != 0 {
		t.Fatalf("clip-rule=evenodd should exclude the overlap, center alpha = %d", c)
	}
}

func TestClipRuleDefaultIsNonZero(t *testing.T) {
	c, _ := renderClip(t, `<path d="M0 0h10v10H0z M3 3h4v4H3z"/>`)
	if c == 0 {
		t.Fatal("default nonzero should keep the overlap filled")
	}
}

func TestClipRuleExplicitNonZero(t *testing.T) {
	c, _ := renderClip(t, `<path clip-rule="nonzero" d="M0 0h10v10H0z M3 3h4v4H3z"/>`)
	if c == 0 {
		t.Fatal("explicit nonzero should keep the overlap filled")
	}
}

func TestClipRuleInheritedFromClipPath(t *testing.T) {
	// clip-rule 写在 <clipPath> 上，子元素继承
	inner := `<svg viewBox="0 0 10 10">
  <defs><clipPath id="c" clip-rule="evenodd"><path d="M0 0h10v10H0z M3 3h4v4H3z"/></clipPath></defs>
  <rect width="10" height="10" fill="red" clip-path="url(#c)"/>
</svg>`
	img, err := Render([]byte(inner), 20)
	if err != nil {
		t.Fatal(err)
	}
	if a := img.NRGBAAt(10, 10).A; a != 0 {
		t.Fatalf("clip-rule on <clipPath> should be inherited, center alpha = %d", a)
	}
}

func TestClipRuleViaCSS(t *testing.T) {
	inner := `<style>#c{clip-rule:evenodd}</style>` // 放在 svg 下、clipPath 之外
	svg := `<svg viewBox="0 0 10 10">` + inner +
		`<defs><clipPath id="c"><path d="M0 0h10v10H0z M3 3h4v4H3z"/></clipPath></defs>` +
		`<rect width="10" height="10" fill="red" clip-path="url(#c)"/></svg>`
	img, err := Render([]byte(svg), 20)
	if err != nil {
		t.Fatal(err)
	}
	if a := img.NRGBAAt(10, 10).A; a != 0 {
		t.Fatalf("clip-rule from CSS should apply, center alpha = %d", a)
	}
}

func TestClipRuleSiblingsDoNotLeak(t *testing.T) {
	// 第一个子元素 evenodd、第二个不带 clip-rule：第二个必须回到 nonzero（只从父继承）
	c, _ := renderClip(t, `<path clip-rule="evenodd" d="M0 0h10v10H0z"/>`+
		`<path d="M3 3h4v4H3z"/>`)
	if c == 0 {
		t.Fatal("second sibling must not inherit the first sibling's clip-rule")
	}
}

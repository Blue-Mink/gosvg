package gosvg

import "testing"

// TestStyleClassSelector 验证 class 选择器。
func TestStyleClassSelector(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><style>.red{fill:#ff0000}</style><rect class="red" width="100" height="100"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if redPx(img) == 0 {
		t.Error("class 选择器应把 rect 填红")
	}
}

// TestStyleTypeSelector 验证类型选择器。
func TestStyleTypeSelector(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><style>rect{fill:#ff0000}</style><rect width="100" height="100"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if redPx(img) == 0 {
		t.Error("类型选择器应把 rect 填红")
	}
}

// TestStyleIDSelector 验证 id 选择器。
func TestStyleIDSelector(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><style>#a{fill:#ff0000}</style><rect id="a" width="100" height="100"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if redPx(img) == 0 {
		t.Error("id 选择器应把 rect 填红")
	}
}

// TestStyleSpecificity 验证 id 特异性高于 class。
func TestStyleSpecificity(t *testing.T) {
	// #a 特异性 (1,0,0) > .red (0,1,0)，fill 应是蓝色
	data := []byte(`<svg viewBox="0 0 100 100"><style>.red{fill:#ff0000}#a{fill:#0000ff}</style><rect id="a" class="red" width="100" height="100"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if redPx(img) != 0 {
		t.Error("id 特异性高于 class，不应是红色")
	}
	if countPx(img, func(r, g, b, a uint8) bool { return a > 0 && b > 200 && r < 100 }) == 0 {
		t.Error("id 规则应填蓝")
	}
}

// TestStyleInlinePriority 验证内联 style 优先于 CSS 规则。
func TestStyleInlinePriority(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><style>.red{fill:#ff0000}</style><rect class="red" style="fill:#0000ff" width="100" height="100"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if redPx(img) != 0 {
		t.Error("内联 style 应覆盖 CSS 规则的红色")
	}
	if countPx(img, func(r, g, b, a uint8) bool { return a > 0 && b > 200 && r < 100 }) == 0 {
		t.Error("内联 style 应填蓝")
	}
}

// TestStyleOverridesPresentation 验证 CSS 规则覆盖 presentation 属性。
func TestStyleOverridesPresentation(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><style>rect{fill:#ff0000}</style><rect fill="#00ff00" width="100" height="100"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if redPx(img) == 0 {
		t.Error("CSS 规则应覆盖 presentation 属性的绿色")
	}
}

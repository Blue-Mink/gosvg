package gosvg

import (
	"math"
	"testing"
)

func TestParseViewportLengthUnits(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want float64
	}{
		{"193mm", 193 * 96.0 / 25.4},
		{"180mm", 180 * 96.0 / 25.4},
		{"1in", 96},
		{"2.54cm", 96},
		{"72pt", 96},
		{"6pc", 96},
		{"101.6q", 96},
		{"100px", 100},
		{"100", 100},
		{"50%", 50},
		{"  193MM  ", 193 * 96.0 / 25.4},
		{"1IN", 96},
		{"1e2", 100},
		{"3.", 3},
		{"-5", -5},
		{"2em", 2},
		{"3rem", 3},
		{"4ex", 4},
		{"5ch", 5},
		{"6vw", 6},
		{"7vh", 7},
		{"8vmin", 8},
		{"9vmax", 9},
		{"12abc", 0},
		{"", 0},
		{"abc", 0},
	} {
		got := parseViewportLength(tc.in, 0)
		if math.Abs(got-tc.want) > 1e-6 {
			t.Errorf("parseViewportLength(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestParseViewportLengthDefault(t *testing.T) {
	if got := parseViewportLength("", 12); got != 12 {
		t.Fatalf("empty should return default, got %v", got)
	}
}

// 端到端：物理单位视口 + 内容坐标，等比缩放后应落在正确位置（与 cairosvg 一致）。
func TestPhysicalUnitViewportPositionsContent(t *testing.T) {
	// 193mm × 180mm → 729.45 × 680.31 px；等比 meet s=256/729.45=0.3509，ty≈8.65
	// rect 用户坐标 x=145..538 → 像素 51..189
	svg := `<svg width="193mm" height="180mm"><rect x="145" y="20" width="393" height="140" fill="red"/></svg>`
	img, err := Render([]byte(svg), 256)
	if err != nil {
		t.Fatal(err)
	}
	if a := img.NRGBAAt(120, 40).A; a == 0 {
		t.Fatal("center of the rect should be filled")
	}
	if a := img.NRGBAAt(10, 40).A; a != 0 {
		t.Fatalf("left area should be empty (got alpha %d)", a)
	}
	if a := img.NRGBAAt(245, 40).A; a != 0 {
		t.Fatalf("right area should be empty (got alpha %d)", a)
	}
}

// 有 viewBox 时走 viewBox 分支，视口 width/height 的单位换算不影响渲染
// （元素几何仍由 parseFloatOr 处理，单位解析是独立的既有缺口）。
func TestViewportUnitsDoNotAffectViewBoxRendering(t *testing.T) {
	a := `<svg viewBox="0 0 20 20" width="20" height="20"><rect width="10" height="10" fill="red"/></svg>`
	b := `<svg viewBox="0 0 20 20" width="20mm" height="20mm"><rect width="10" height="10" fill="red"/></svg>`
	i1, err1 := Render([]byte(a), 32)
	i2, err2 := Render([]byte(b), 32)
	if err1 != nil || err2 != nil {
		t.Fatalf("render err: %v / %v", err1, err2)
	}
	for i := 0; i < len(i1.Pix); i++ {
		if i1.Pix[i] != i2.Pix[i] {
			t.Fatal("viewBox path must not depend on viewport physical units")
		}
	}
}

// 相对单位视口必须按「数值 1:1 近似」正常渲染，不能退化成 0 尺寸。
func TestViewportRelativeUnitsRender(t *testing.T) {
	svg := `<svg width="100em" height="80em"><rect width="100" height="80" fill="red"/></svg>`
	img, err := Render([]byte(svg), 64)
	if err != nil {
		t.Fatalf("relative-unit viewport must render, got %v", err)
	}
	if a := img.NRGBAAt(32, 32).A; a == 0 {
		t.Fatal("center should be filled")
	}
}

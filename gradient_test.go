package gosvg

import (
	"testing"
)

// TestLinearGradient 验证 linearGradient（默认 objectBoundingBox）左红右蓝。
func TestLinearGradient(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="0"><stop offset="0" stop-color="#ff0000"/><stop offset="1" stop-color="#0000ff"/></linearGradient></defs><rect width="100" height="100" fill="url(#g)"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	left := img.NRGBAAt(10, 50)
	mid := img.NRGBAAt(50, 50)
	right := img.NRGBAAt(90, 50)
	if !(left.R > left.B) {
		t.Errorf("left 应偏红: %v", left)
	}
	if !(right.B > right.R) {
		t.Errorf("right 应偏蓝: %v", right)
	}
	// 中间应大致红蓝混合（R≈B，均>0）
	if mid.R < 100 || mid.B < 100 || absInt(int(mid.R)-int(mid.B)) > 40 {
		t.Errorf("mid 应偏紫(红蓝混合): %v", mid)
	}
}

// TestRadialGradient 验证 radialGradient（userSpaceOnUse）中心白边缘黑。
func TestRadialGradient(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><defs><radialGradient id="r" gradientUnits="userSpaceOnUse" cx="50" cy="50" r="50"><stop offset="0" stop-color="#ffffff"/><stop offset="1" stop-color="#000000"/></radialGradient></defs><rect width="100" height="100" fill="url(#r)"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	center := img.NRGBAAt(50, 50)
	edge := img.NRGBAAt(5, 50)
	if !(center.R > 200 && center.G > 200 && center.B > 200) {
		t.Errorf("radial 中心应偏白: %v", center)
	}
	if !(edge.R < 60 && edge.G < 60 && edge.B < 60) {
		t.Errorf("radial 边缘应偏黑: %v", edge)
	}
}

// TestGradientHrefInheritance 验证 linearGradient 通过 href 继承 stops。
func TestGradientHrefInheritance(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><defs><linearGradient id="base"><stop offset="0" stop-color="#00ff00"/><stop offset="1" stop-color="#0000ff"/></linearGradient><linearGradient id="g" href="#base" x1="0" y1="0" x2="1" y2="0"/></defs><rect width="100" height="100" fill="url(#g)"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	left := img.NRGBAAt(10, 50)
	right := img.NRGBAAt(90, 50)
	if !(left.G > 150 && left.R < 90) {
		t.Errorf("left 应偏绿(继承 base stop0=#00ff00): %v", left)
	}
	if !(right.B > 150 && right.R < 90) {
		t.Errorf("right 应偏蓝(继承 base stop1=#0000ff): %v", right)
	}
}

// TestGradientStopsInterpolate 验证三个 stop 的中间插值。
func TestGradientStopsInterpolate(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="0"><stop offset="0" stop-color="#ff0000"/><stop offset="0.5" stop-color="#00ff00"/><stop offset="1" stop-color="#0000ff"/></linearGradient></defs><rect width="100" height="100" fill="url(#g)"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// 中间(50,50) 应偏绿（offset 0.5 = #00ff00）
	mid := img.NRGBAAt(50, 50)
	if !(mid.G > 200 && mid.R < 90 && mid.B < 90) {
		t.Errorf("mid 应偏绿(offset 0.5): %v", mid)
	}
}

// TestGradientStopOpacity 验证 stop-opacity 生效。
func TestGradientStopOpacity(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="0"><stop offset="0" stop-color="#000000" stop-opacity="1"/><stop offset="1" stop-color="#000000" stop-opacity="0"/></linearGradient></defs><rect width="100" height="100" fill="url(#g)"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	left := img.NRGBAAt(10, 50)
	right := img.NRGBAAt(90, 50)
	if left.A < 200 {
		t.Errorf("left 应不透明(stop-opacity=1): %v", left)
	}
	if right.A > 40 {
		t.Errorf("right 应透明(stop-opacity=0): %v", right)
	}
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

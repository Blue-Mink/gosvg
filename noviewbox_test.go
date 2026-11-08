package gosvg

import "testing"

// 无 viewBox 时必须保持 width/height 的宽高比并居中（xMidYMid meet），
// 不能非等比拉伸。真实语料：resizeHorizontal.svg（6×26）此前 IoU = 0。
func TestNoViewBoxPreservesAspectRatio(t *testing.T) {
	svg := `<svg width="6" height="26"><rect width="6" height="26" fill="red"/></svg>`
	img, err := Render([]byte(svg), 256)
	if err != nil {
		t.Fatal(err)
	}
	// 等比缩放：s = min(256/6, 256/26) = 9.846 → 内容 59×256，水平居中 x≈98..157
	if a := img.NRGBAAt(128, 128).A; a == 0 {
		t.Fatal("center should be filled")
	}
	if a := img.NRGBAAt(10, 128).A; a != 0 {
		t.Fatalf("left margin should be transparent (got alpha %d)", a)
	}
	if a := img.NRGBAAt(246, 128).A; a != 0 {
		t.Fatalf("right margin should be transparent (got alpha %d)", a)
	}
	if a := img.NRGBAAt(128, 0).A; a == 0 {
		t.Fatal("content should span the full height")
	}
}

func TestNoViewBoxNoneStillStretches(t *testing.T) {
	// preserveAspectRatio="none" 是显式的拉伸要求，保持旧行为。
	svg := `<svg width="6" height="26" preserveAspectRatio="none"><rect width="6" height="26" fill="red"/></svg>`
	img, err := Render([]byte(svg), 256)
	if err != nil {
		t.Fatal(err)
	}
	if a := img.NRGBAAt(10, 128).A; a == 0 {
		t.Fatal("preserveAspectRatio=none should stretch to fill")
	}
}

func TestNoViewBoxWithoutDimensionsIsIdentity(t *testing.T) {
	// width/height 缺失时保持 1:1（回归保护）
	svg := `<svg><rect x="10" y="10" width="20" height="20" fill="red"/></svg>`
	img, err := Render([]byte(svg), 64)
	if err != nil {
		t.Fatal(err)
	}
	if a := img.NRGBAAt(20, 20).A; a == 0 {
		t.Fatal("1:1 mapping should keep the rect at 10..30")
	}
}

func TestViewBoxUnchangedByNewBranch(t *testing.T) {
	// 有 viewBox 时走原逻辑（回归保护，golden 覆盖）
	svg := `<svg viewBox="0 0 10 20"><rect width="10" height="20" fill="red"/></svg>`
	img, err := Render([]byte(svg), 256)
	if err != nil {
		t.Fatal(err)
	}
	if a := img.NRGBAAt(128, 128).A; a == 0 {
		t.Fatal("viewBox path must still fill")
	}
}

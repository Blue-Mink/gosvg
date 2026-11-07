package gosvg

import "testing"

// TestPreserveAspectRatioNone 验证 preserveAspectRatio="none" 拉伸填充。
func TestPreserveAspectRatioNone(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 200" preserveAspectRatio="none"><rect width="100" height="200" fill="red"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if n := countAlpha(img); n != 100*100 {
		t.Errorf("none 应拉伸填满：got %d, want %d", n, 100*100)
	}
}

// TestPreserveAspectRatioMeetDefault 验证默认 xMidYMid meet（居中留白）。
func TestPreserveAspectRatioMeetDefault(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 200"><rect width="100" height="200" fill="red"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// meet：s=min(1,0.5)=0.5，viewBox 映射 50×100，居中左右留白各 25。
	if n := countAlpha(img); n != 50*100 {
		t.Errorf("meet 应居中留白：got %d, want %d", n, 50*100)
	}
}

// TestPreserveAspectRatioAlign 验证 xMinYMin / xMaxYMax 对齐。
func TestPreserveAspectRatioAlign(t *testing.T) {
	// viewBox 50×100 → 100×100 视口，meet s=1，内容 50 宽。
	minImg, err := Render([]byte(`<svg viewBox="0 0 50 100" preserveAspectRatio="xMinYMin meet"><rect width="50" height="100" fill="red"/></svg>`), 100)
	if err != nil {
		t.Fatalf("xMinYMin render: %v", err)
	}
	maxImg, err := Render([]byte(`<svg viewBox="0 0 50 100" preserveAspectRatio="xMaxYMax meet"><rect width="50" height="100" fill="red"/></svg>`), 100)
	if err != nil {
		t.Fatalf("xMaxYMax render: %v", err)
	}
	// xMinYMin：红在左（x<50）
	if _, _, _, a := minImg.At(25, 50).RGBA(); a == 0 {
		t.Error("xMinYMin 左侧应有红")
	}
	if _, _, _, a := minImg.At(75, 50).RGBA(); a != 0 {
		t.Error("xMinYMin 右侧应透明")
	}
	// xMaxYMax：红在右（x>=50）
	if _, _, _, a := maxImg.At(25, 50).RGBA(); a != 0 {
		t.Error("xMaxYMax 左侧应透明")
	}
	if _, _, _, a := maxImg.At(75, 50).RGBA(); a == 0 {
		t.Error("xMaxYMax 右侧应有红")
	}
}

// TestPreserveAspectRatioSlice 验证 slice 裁剪（缩放至覆盖，超界裁剪）。
func TestPreserveAspectRatioSlice(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 200" preserveAspectRatio="xMidYMid slice"><rect width="100" height="200" fill="red"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// slice：s=max(1,0.5)=1，viewBox 100×200 → 100×200，x 方向填满 100。
	if _, _, _, a := img.At(10, 50).RGBA(); a == 0 {
		t.Error("slice 应 x 方向填满（无留白）")
	}
}

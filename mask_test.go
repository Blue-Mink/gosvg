package gosvg

import (
	"testing"
)

// TestMaskContentUnitsObjectBoundingBox 验证 maskContentUnits=objectBoundingBox
// 时子元素坐标按 0..1 归一化映射到被遮罩元素包围盒。
func TestMaskContentUnitsObjectBoundingBox(t *testing.T) {
	// mask 子元素 rect(0,0,1,1) 覆盖整个 bbox，被遮罩元素是 (20,20,60,60)。
	data := []byte(`<svg viewBox="0 0 100 100"><mask id="m" maskContentUnits="objectBoundingBox"><rect x="0" y="0" width="1" height="1" fill="#fff"/></mask><rect x="20" y="20" width="60" height="60" fill="#f00" mask="url(#m)"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// 中心(50,50) 在被遮罩元素内，mask 覆盖全 bbox → 可见
	if _, _, _, a := img.At(50, 50).RGBA(); a == 0 {
		t.Error("maskContentUnits=objectBoundingBox 全 bbox 覆盖时中心应可见")
	}
	// 外圈(5,5) 在被遮罩元素外 → 透明
	if _, _, _, a := img.At(5, 5).RGBA(); a != 0 {
		t.Error("被遮罩元素外应透明")
	}
}

// TestMaskContentUnitsPartial 验证 objectBoundingBox 的部分覆盖（左半可见右半透明）。
func TestMaskContentUnitsPartial(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><mask id="m" maskContentUnits="objectBoundingBox"><rect x="0" y="0" width="0.5" height="1" fill="#fff"/></mask><rect x="20" y="20" width="60" height="60" fill="#f00" mask="url(#m)"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// 左半(35,50) 可见
	if _, _, _, a := img.At(35, 50).RGBA(); a == 0 {
		t.Error("mask 覆盖左半 bbox，左半应可见")
	}
	// 右半(65,50) 透明
	if _, _, _, a := img.At(65, 50).RGBA(); a != 0 {
		t.Error("mask 未覆盖右半 bbox，右半应透明")
	}
}

// TestMaskTransform 验证 mask 元素自身的 transform 生效。
func TestMaskTransform(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><mask id="m" transform="translate(50,0)"><rect x="0" y="0" width="50" height="100" fill="#fff"/></mask><rect width="100" height="100" fill="#f00" mask="url(#m)"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// mask 子元素经 translate(50,0) 后覆盖右半 → 右半可见
	if _, _, _, a := img.At(75, 50).RGBA(); a == 0 {
		t.Error("mask transform 后右半应可见")
	}
	// 左半透明
	if _, _, _, a := img.At(25, 50).RGBA(); a != 0 {
		t.Error("mask transform 后左半应透明")
	}
}

// TestMaskUnitsRegion 验证 maskUnits=userSpaceOnUse 的区域裁剪。
func TestMaskUnitsRegion(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><mask id="m" maskUnits="userSpaceOnUse" x="0" y="0" width="50" height="100"><rect x="0" y="0" width="100" height="100" fill="#fff"/></mask><rect width="100" height="100" fill="#f00" mask="url(#m)"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// mask 区域 (0,0)-(50,100)，子元素覆盖全区但裁剪到左半 → 左半可见
	if _, _, _, a := img.At(25, 50).RGBA(); a == 0 {
		t.Error("maskUnits 区域左半应可见")
	}
	// 右半在 mask 区域外 → 透明
	if _, _, _, a := img.At(75, 50).RGBA(); a != 0 {
		t.Error("maskUnits 区域右半应透明")
	}
}

// TestMaskUnitsObjectBoundingBox 验证 maskUnits=objectBoundingBox 的区域裁剪
// （归一化区域映射到 bbox）。mask 子元素用默认 userSpaceOnUse 覆盖全用户空间。
func TestMaskUnitsObjectBoundingBox(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><mask id="m" maskUnits="objectBoundingBox" x="0" y="0" width="0.5" height="1"><rect x="0" y="0" width="100" height="100" fill="#fff"/></mask><rect x="20" y="20" width="60" height="60" fill="#f00" mask="url(#m)"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// mask 区域 = bbox 左半，子元素覆盖全用户空间 → 左半可见
	if _, _, _, a := img.At(30, 50).RGBA(); a == 0 {
		t.Error("maskUnits=objectBoundingBox 左半应可见")
	}
	if _, _, _, a := img.At(70, 50).RGBA(); a != 0 {
		t.Error("maskUnits=objectBoundingBox 右半应透明")
	}
}

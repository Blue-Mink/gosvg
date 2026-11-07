package gosvg

import "testing"

// TestEvenOddWinding 验证 fill-rule="evenodd" 真镂空。
// 之前 ScannerGV.SetWinding 是 no-op（底层 vector 光栅器仅 non-zero），
// 现已实现 even-odd accumulate。
func TestEvenOddWinding(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><path fill="#f00" fill-rule="evenodd" d="M0 0h100v100H0zM20 20h60v60H20z"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if _, _, _, a := img.At(50, 50).RGBA(); a != 0 {
		t.Error("evenodd 内圈应镂空")
	}
	if _, _, _, a := img.At(10, 50).RGBA(); a == 0 {
		t.Error("evenodd 外圈应填充")
	}
}

// TestEvenOddVsNonZero 验证 evenodd 与 nonzero（默认）对同路径的差异。
func TestEvenOddVsNonZero(t *testing.T) {
	const path = `M0 0h100v100H0zM20 20h60v60H20z`
	render := func(rule string) uint32 {
		data := []byte(`<svg viewBox="0 0 100 100"><path fill="#f00" fill-rule="` + rule + `" d="` + path + `"/></svg>`)
		img, _ := Render(data, 100)
		_, _, _, a := img.At(50, 50).RGBA()
		return a
	}
	if render("evenodd") != 0 {
		t.Error("evenodd 中心应镂空")
	}
	if render("nonzero") == 0 {
		t.Error("nonzero 中心应填充（同向 winding=2）")
	}
}

// TestEvenOddTripleWinding 验证 winding=3（奇数）时 evenodd 填充、winding=2 镂空。
func TestEvenOddTripleWinding(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><path fill="#f00" fill-rule="evenodd" d="M0 0h100v100H0zM20 20h60v60H20zM40 40h20v20H40z"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if _, _, _, a := img.At(50, 50).RGBA(); a == 0 {
		t.Error("evenodd winding=3（奇数）应填充")
	}
	if _, _, _, a := img.At(30, 50).RGBA(); a != 0 {
		t.Error("evenodd winding=2（偶数）应镂空")
	}
	if _, _, _, a := img.At(10, 50).RGBA(); a == 0 {
		t.Error("evenodd winding=1（奇数）应填充")
	}
}

// TestEvenOddGradient 验证 evenodd 与 gradient 组合。
func TestEvenOddGradient(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><defs><linearGradient id="g"><stop offset="0" stop-color="#ff0000"/><stop offset="1" stop-color="#0000ff"/></linearGradient></defs><path fill="url(#g)" fill-rule="evenodd" d="M0 0h100v100H0zM20 20h60v60H20z"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if _, _, _, a := img.At(50, 50).RGBA(); a != 0 {
		t.Error("evenodd+gradient 内圈应镂空")
	}
	if _, _, _, a := img.At(10, 50).RGBA(); a == 0 {
		t.Error("evenodd+gradient 外圈应填充")
	}
}

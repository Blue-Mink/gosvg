package gosvg

import "testing"

func TestParseColorRGBIntegerComponents(t *testing.T) {
	// SVG/CSS 规范：rgb() 无百分号分量是 0..255 整数。此前误用 clamp01(f)*255，
	// 把 128 夹到 1 → 255，导致所有中间整数值渲染成白色（只有 0/255 碰巧对）。
	cases := []struct {
		in   string
		want [3]uint8
	}{
		{"rgb(128,64,32)", [3]uint8{128, 64, 32}},
		{"rgb(1,2,3)", [3]uint8{1, 2, 3}},
		{"rgb(0,0,0)", [3]uint8{0, 0, 0}},
		{"rgb(255,255,255)", [3]uint8{255, 255, 255}},
		{"rgb(200 100 50)", [3]uint8{200, 100, 50}}, // 空格分隔
		{"rgb(127.5 0 0)", [3]uint8{128, 0, 0}},     // 小数四舍五入
		{"rgb(300,-5,0)", [3]uint8{255, 0, 0}},      // 越界夹取
		{"rgb(50%,25%,10%)", [3]uint8{127, 63, 25}}, // 百分比语义不变
	}
	for _, tc := range cases {
		got, ok := parseColor(tc.in)
		if !ok {
			t.Errorf("parseColor(%q) 失败", tc.in)
			continue
		}
		if got.R != tc.want[0] || got.G != tc.want[1] || got.B != tc.want[2] {
			t.Errorf("parseColor(%q) = (%d,%d,%d), want (%d,%d,%d)", tc.in,
				got.R, got.G, got.B, tc.want[0], tc.want[1], tc.want[2])
		}
	}
}

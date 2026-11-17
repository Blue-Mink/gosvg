package gosvg

import (
	"math"
	"testing"
)

func TestParseTransformSkewStandardCase(t *testing.T) {
	// SVG 规范函数名是 skewX/skewY（大写 X/Y）。此前 name 未转小写、switch
	// 只认非标准的小写 skewx/skewy，导致标准写法返回单位矩阵（变换失效）。
	deg := 30.0
	want := math.Tan(deg * math.Pi / 180)

	t.Run("skewX", func(t *testing.T) {
		m := parseTransform("skewX(30)")
		if math.Abs(m.C-want) > 1e-9 || m.B != 0 {
			t.Errorf("skewX(30) = %+v, want C=tan30≈%.6f B=0", m, want)
		}
	})
	t.Run("skewY", func(t *testing.T) {
		m := parseTransform("skewY(30)")
		if math.Abs(m.B-want) > 1e-9 || m.C != 0 {
			t.Errorf("skewY(30) = %+v, want B=tan30≈%.6f C=0", m, want)
		}
	})
	t.Run("lowercase still works", func(t *testing.T) {
		if m := parseTransform("skewx(30)"); math.Abs(m.C-want) > 1e-9 {
			t.Errorf("skewx(30) C=%.6f, want %.6f", m.C, want)
		}
	})
	t.Run("other functions unaffected", func(t *testing.T) {
		m := parseTransform("translate(10,20)")
		if m.E != 10 || m.F != 20 {
			t.Errorf("translate(10,20) = %+v", m)
		}
	})
}

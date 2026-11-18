package gosvg

import (
	"strconv"
	"testing"
)

// 测水平线描边的垂直厚度（统计有多少行含不透明像素）。
func strokeThickness(t *testing.T, vectorEffect string, scale float64) int {
	ve := ""
	if vectorEffect != "" {
		ve = ` vector-effect="` + vectorEffect + `"`
	}
	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="400" height="400" viewBox="0 0 400 400">` +
		`<path d="M150 100 H250" fill="none" stroke="black" stroke-width="10"` + ve +
		` transform="scale(` + ftoa(scale) + `)"/></svg>`
	doc, err := Parse([]byte(svg))
	if err != nil {
		t.Fatalf("Parse err: %v", err)
	}
	img, err := doc.Render(400, 400)
	if err != nil {
		t.Fatalf("Render err: %v", err)
	}
	rows := 0
	for y := 0; y < 400; y++ {
		hit := false
		for x := 0; x < 400; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 {
				hit = true
				break
			}
		}
		if hit {
			rows++
		}
	}
	return rows
}

func ftoa(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func TestVectorEffectNonScalingStroke(t *testing.T) {
	// 普通描边：scale(2) 时厚度约翻倍。
	normal1 := strokeThickness(t, "", 1)
	normal2 := strokeThickness(t, "", 2)
	t.Logf("normal  scale1=%d scale2=%d", normal1, normal2)
	if normal2 < normal1*15/10 {
		t.Errorf("普通描边未随 scale 放大: s1=%d s2=%d", normal1, normal2)
	}

	// non-scaling：scale(2) / scale(0.5) 厚度应基本恒定（≈scale1 的厚度）。
	ns1 := strokeThickness(t, "non-scaling-stroke", 1)
	ns2 := strokeThickness(t, "non-scaling-stroke", 2)
	nsHalf := strokeThickness(t, "non-scaling-stroke", 0.5)
	t.Logf("non-scaling scale1=%d scale2=%d scale0.5=%d", ns1, ns2, nsHalf)
	tol := 3
	if ns2 < ns1-tol || ns2 > ns1+tol {
		t.Errorf("non-scaling-stroke scale2=%d 偏离 scale1=%d (tol=%d)", ns2, ns1, tol)
	}
	if nsHalf < ns1-tol || nsHalf > ns1+tol {
		t.Errorf("non-scaling-stroke scale0.5=%d 偏离 scale1=%d (tol=%d)", nsHalf, ns1, tol)
	}

	// vector-effect:none 应与普通描边一致（默认值无副作用）。
	none2 := strokeThickness(t, "none", 2)
	if none2 != normal2 {
		t.Errorf("vector-effect:none 与普通描边不同: none=%d normal=%d", none2, normal2)
	}
}

package gosvg

import "testing"

func almost(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 1e-6
}

func rangesEqual(a, b [][2]float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !almost(a[i][0], b[i][0]) || !almost(a[i][1], b[i][1]) {
			return false
		}
	}
	return true
}

func TestParseDashArray(t *testing.T) {
	if _, ok := parseDashArray("none"); ok {
		t.Error("'none' should not be a dash array")
	}
	if _, ok := parseDashArray(""); ok {
		t.Error("empty should not be a dash array")
	}
	d, ok := parseDashArray("5 5")
	if !ok || len(d) != 2 || d[0] != 5 || d[1] != 5 {
		t.Errorf("parseDashArray('5 5') = %v, %v", d, ok)
	}
	d, ok = parseDashArray("5,5,10")
	if !ok || len(d) != 3 || d[2] != 10 {
		t.Errorf("parseDashArray('5,5,10') = %v, %v", d, ok)
	}
	// 非法值 → false
	if _, ok := parseDashArray("5 foo"); ok {
		t.Error("'5 foo' should fail")
	}
}

func TestNormalizeDashes(t *testing.T) {
	if normalizeDashes([]float64{}) != nil {
		t.Error("empty -> nil")
	}
	// 奇数长度复制成偶数
	got := normalizeDashes([]float64{20, 10, 5})
	want := []float64{20, 10, 5, 20, 10, 5}
	if len(got) != len(want) {
		t.Fatalf("normalize odd len = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("normalize odd len[%d] = %v, want %v", i, got[i], want[i])
		}
	}
	// 全零 → nil
	if normalizeDashes([]float64{0, 0}) != nil {
		t.Error("all-zero -> nil")
	}
	// 负值 → nil
	if normalizeDashes([]float64{5, -1}) != nil {
		t.Error("negative -> nil")
	}
}

func TestDashRangesSimple(t *testing.T) {
	r := dashRanges(0, []float64{20, 20}, 100)
	want := [][2]float64{{0, 20}, {40, 60}, {80, 100}}
	if !rangesEqual(r, want) {
		t.Errorf("dashRanges(0, [20,20], 100) = %v, want %v", r, want)
	}
}

func TestDashRangesOffset(t *testing.T) {
	// offset=10：第一个 dash 从 -10 到 10，路径内有效部分 [0,10]。
	r := dashRanges(10, []float64{20, 20}, 100)
	want := [][2]float64{{0, 10}, {30, 50}, {70, 90}}
	if !rangesEqual(r, want) {
		t.Errorf("dashRanges(10, [20,20], 100) = %v, want %v", r, want)
	}
}

func TestDashRangesOddDashes(t *testing.T) {
	// [10, 5] 奇数长度已在 normalize 后为偶数；这里直接测 dashRanges 用偶数数组。
	// dash 10 gap 5：dash 段 [0,10],[15,25],[30,40]... 到 100。
	r := dashRanges(0, []float64{10, 5}, 20)
	want := [][2]float64{{0, 10}, {15, 20}}
	if !rangesEqual(r, want) {
		t.Errorf("dashRanges(0, [10,5], 20) = %v, want %v", r, want)
	}
}

func TestDashSegmentsHorizontal(t *testing.T) {
	pts := []Point{{0, 0}, {100, 0}}
	segs := dashSegments(pts, 0, []float64{20, 20}, false)
	if len(segs) != 3 {
		t.Fatalf("dashSegments count = %d, want 3", len(segs))
	}
	// 每段应为 [x0,0],[x1,0]
	if !almost(segs[0][0].X, 0) || !almost(segs[0][1].X, 20) {
		t.Errorf("seg0 = %v, want [0,20]", segs[0])
	}
	if !almost(segs[1][0].X, 40) || !almost(segs[1][1].X, 60) {
		t.Errorf("seg1 = %v, want [40,60]", segs[1])
	}
	if !almost(segs[2][0].X, 80) || !almost(segs[2][1].X, 100) {
		t.Errorf("seg2 = %v, want [80,100]", segs[2])
	}
}

func TestDashSegmentsPolylineVertex(t *testing.T) {
	// 折线带中间顶点：dash 跨越顶点时应包含该顶点。
	pts := []Point{{0, 0}, {20, 0}, {40, 0}, {60, 0}}
	segs := dashSegments(pts, 0, []float64{30, 30}, false)
	// 总长 60，dash 30 gap 30 → 仅一个 dash [0,30]，gap [30,60]。
	if len(segs) != 1 {
		t.Fatalf("dashSegments count = %d, want 1", len(segs))
	}
	// seg0 = [0,30]：含顶点 (20,0)，终点 (30,0)。
	if len(segs[0]) != 3 {
		t.Fatalf("seg0 should have 3 points (0,20,30), got %d: %v", len(segs[0]), segs[0])
	}
	if !almost(segs[0][1].X, 20) || !almost(segs[0][2].X, 30) {
		t.Errorf("seg0 = %v, want [0,20,30]", segs[0])
	}
}

func TestDashClosedRect(t *testing.T) {
	// 正方形 40×40 环（展开折线，含首尾闭合点），周长 160。
	pts := []Point{{0, 0}, {0, 40}, {40, 40}, {40, 0}, {0, 0}}
	segs := dashSegments(pts, 0, []float64{20, 20}, true)
	// dash [0,20],[40,60],[80,100],[120,140] → 4 段。
	if len(segs) != 4 {
		t.Fatalf("closed dashSegments count = %d, want 4", len(segs))
	}
}

func TestDashClosedWrapAround(t *testing.T) {
	// 跨起点的 dash：offset=10，dash 20 gap 20，周长 160。
	// dash ranges = [0,10],[30,50],[70,90],[110,130],[150,160]，
	// 首段 [0,10] 与末段 [150,160] 是同一 dash 被起点切断 → 合并，共 4 段。
	pts := []Point{{0, 0}, {0, 40}, {40, 40}, {40, 0}, {0, 0}}
	segs := dashSegments(pts, 10, []float64{20, 20}, true)
	if len(segs) != 4 {
		t.Fatalf("wrap-around closed dash count = %d, want 4; got %v", len(segs), segs)
	}
}

// TestRenderDashLine 端到端：水平虚线。
func TestRenderDashLine(t *testing.T) {
	data := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 20"><line x1="0" y1="10" x2="100" y2="10" stroke="red" stroke-width="2" stroke-dasharray="20 20"/></svg>`)
	img, err := RenderTo(data, 100, 20)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// y=10 行：dash 段 [0,20],[40,60],[80,100] 有红像素；gap [20,40],[60,80] 无。
	redAt := func(x int) bool {
		r, g, b, a := img.At(x, 10).RGBA()
		if a == 0 {
			return false
		}
		return r>>8 > 150 && g>>8 < 90 && b>>8 < 90
	}
	if !redAt(10) || !redAt(50) || !redAt(90) {
		t.Error("expected dash pixels at x=10/50/90")
	}
	if redAt(30) || redAt(70) {
		t.Error("expected gaps at x=30/70")
	}
}

// TestRenderDashOffset 端到端：dashoffset 平移。
func TestRenderDashOffset(t *testing.T) {
	data := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 20"><line x1="0" y1="10" x2="100" y2="10" stroke="red" stroke-width="2" stroke-dasharray="20 20" stroke-dashoffset="10"/></svg>`)
	img, err := RenderTo(data, 100, 20)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	redAt := func(x int) bool {
		r, _, b, a := img.At(x, 10).RGBA()
		if a == 0 {
			return false
		}
		return r>>8 > 150 && b>>8 < 90
	}
	// offset=10: dash [0,10],[30,50],[70,90]
	if !redAt(5) || !redAt(40) || !redAt(80) {
		t.Error("expected dash pixels at x=5/40/80 with offset=10")
	}
	if redAt(20) || redAt(60) {
		t.Error("expected gaps at x=20/60 with offset=10")
	}
}

package gosvg

import (
	"bytes"
	"image"
	"strconv"
	"testing"
)

// 本文件给 v0.3.0 / PR #29 新增的 <text> 描边能力补回归保护。
// 该能力此前只在仓库外的 117 文件 cairosvg IoU 子集里被间接验证，
// 仓库内既无单测、golden 也无含 text+stroke 的样例（见 testdata/text_stroke.svg）。
// 断言按「有/无像素、相对差异」写，不锁具体像素数——避免字体/AA 微调就误报。

// textStrokeSVG 构造一个最小的 text 描边样例。viewBox 与输出同尺寸，
// 避免视口缩放把 stroke-width 变成亚像素。
func textStrokeSVG(fill, stroke string, strokeWidth float64, extra string) []byte {
	return []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 200 100" ` +
		`font-size="48" font-family="Go Regular" ` + extra + `>` +
		`<text x="20" y="70" fill="` + fill + `" stroke="` + stroke +
		`" stroke-width="` + strconv.FormatFloat(strokeWidth, 'f', -1, 64) + `">SVG</text></svg>`)
}

// TestTextStrokeRendersWithoutFill 统计 alpha>0 的像素数（与 golden_test.go 的 hasAlpha 口径一致）。
func countOpaque(img *image.NRGBA) int {
	n := 0
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] != 0 {
			n++
		}
	}
	return n
}

// pixelDiff 统计两张同尺寸图中 RGBA 任意通道不同的像素数。
func pixelDiff(a, b *image.NRGBA) int {
	if a.Bounds() != b.Bounds() || len(a.Pix) != len(b.Pix) {
		return -1
	}
	n := 0
	for i := 0; i < len(a.Pix); i += 4 {
		if a.Pix[i] != b.Pix[i] || a.Pix[i+1] != b.Pix[i+1] ||
			a.Pix[i+2] != b.Pix[i+2] || a.Pix[i+3] != b.Pix[i+3] {
			n++
		}
	}
	return n
}

// TestTextStrokeRendersWithoutFill 是 P0 的核心回归：fill:none + stroke 时
// 文字必须被画出来。v0.3.0 之前这一整块是空的（描边不参与 text 绘制）。
func TestTextStrokeRendersWithoutFill(t *testing.T) {
	img, err := Render(textStrokeSVG("none", "#000000", 2, ""), 200)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if n := countOpaque(img); n == 0 {
		t.Fatal("fill=none + stroke 的文字渲染结果全透明：text stroke 未生效（PR #29 回归）")
	}
}

// TestTextStrokeVsFillOnly 描边-only 与填充-only 必须是两张不同的图。
// 注意：不断言「描边像素更少」——实测 font-size=48 / stroke-width=2 时
// 描边轮廓（288px）比 font.Drawer 的字形填充（148px）更大，因为描边是轮廓
// 外扩的环、且两者走完全不同的光栅化路径。只锁「都非空且逐像素不同」。
func TestTextStrokeVsFillOnly(t *testing.T) {
	strokeOnly, err := Render(textStrokeSVG("none", "#000000", 2, ""), 200)
	if err != nil {
		t.Fatalf("render stroke-only: %v", err)
	}
	fillOnly, err := Render(textStrokeSVG("#000000", "none", 2, ""), 200)
	if err != nil {
		t.Fatalf("render fill-only: %v", err)
	}
	ns, nf := countOpaque(strokeOnly), countOpaque(fillOnly)
	if ns == 0 || nf == 0 {
		t.Fatalf("样例不应全透明: stroke=%d fill=%d", ns, nf)
	}
	if d := pixelDiff(strokeOnly, fillOnly); d <= 0 {
		t.Errorf("描边-only 与填充-only 输出应不同，diff=%d", d)
	}
}

// TestTextStrokeWidthChangesOutput stroke-width 必须真的进入设备空间描边管线
// （strokeDevicePath 的 halfWidth = strokeWidth × scale / 2）。
func TestTextStrokeWidthChangesOutput(t *testing.T) {
	thin, err := Render(textStrokeSVG("none", "#000000", 1, ""), 200)
	if err != nil {
		t.Fatalf("render thin: %v", err)
	}
	thick, err := Render(textStrokeSVG("none", "#000000", 4, ""), 200)
	if err != nil {
		t.Fatalf("render thick: %v", err)
	}
	n1, n4 := countOpaque(thin), countOpaque(thick)
	if n1 == 0 || n4 == 0 {
		t.Fatalf("样例不应全透明: width1=%d width4=%d", n1, n4)
	}
	if n4 <= n1 {
		t.Errorf("更粗的描边应覆盖更多像素: width1=%d width4=%d", n1, n4)
	}
}

// TestTextStrokePlusFill 同时给 fill 和 stroke 时，输出应同时区别于
// 「只填充」和「只描边」（证明两条路径都跑到了，且不是互相覆盖成同一张图）。
// 注意：当前实现无条件「先描边、后填充」，不解析 paint-order 属性；
// 这里只断言组合结果与两个单路径都不同，不假定 paint-order 语义。
func TestTextStrokePlusFill(t *testing.T) {
	both, err := Render(textStrokeSVG("#ff0000", "#0000ff", 3, ""), 200)
	if err != nil {
		t.Fatalf("render both: %v", err)
	}
	fillOnly, _ := Render(textStrokeSVG("#ff0000", "none", 3, ""), 200)
	strokeOnly, _ := Render(textStrokeSVG("none", "#0000ff", 3, ""), 200)
	if countOpaque(both) == 0 {
		t.Fatal("fill+stroke 结果全透明")
	}
	if d := pixelDiff(both, fillOnly); d <= 0 {
		t.Errorf("fill+stroke 应与只填充不同（描边应可见），diff=%d", d)
	}
	if d := pixelDiff(both, strokeOnly); d <= 0 {
		t.Errorf("fill+stroke 应与只描边不同（填充应可见），diff=%d", d)
	}
}

// TestTextStrokeNoStrokeDoesNotRender 显式 stroke="none" / 空文本 / 零描边宽度
// 必须不画任何像素，且不应 panic（防 nil geom 回归）。
//
// 注：font-size="0" 已实证为正确行为（不在本用例里）——text.go:30 的
// parseLengthOr(n.attr("font-size"), 12) 在 font-size="0" 时返回 0
// （parseLength("0")=(0,true)，不回退），drawTextSpan 的 fontSize<=0 分支跳过
// 绘制，实测渲染 0 个不透明像素。此前注释称「回退默认 12 照常绘制」系对
// parseLengthOr 的错误推断，已据实更正（纯注释，不改行为）。
func TestTextStrokeNoStrokeDoesNotRender(t *testing.T) {
	for _, tc := range []struct {
		name string
		svg  string
	}{
		{"empty text", `<svg viewBox="0 0 200 100" font-size="48"><text x="20" y="70" fill="none" stroke="black" stroke-width="2"></text></svg>`},
		{"zero stroke-width", `<svg viewBox="0 0 200 100" font-size="48"><text x="20" y="70" fill="none" stroke="black" stroke-width="0">SVG</text></svg>`},
		{"stroke none", `<svg viewBox="0 0 200 100" font-size="48"><text x="20" y="70" fill="none" stroke="none" stroke-width="2">SVG</text></svg>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			img, err := Render([]byte(tc.svg), 200)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			if n := countOpaque(img); n != 0 {
				t.Errorf("预期全透明，得到 %d 个不透明像素", n)
			}
		})
	}
}

// TestTextStrokeDeterministic 同一输入重复渲染必须逐字节一致：
// strokeTextSpan 复用 ctx.textPath 缓冲，若 Clear 漏了会累积上一次的字形。
func TestTextStrokeDeterministic(t *testing.T) {
	data := textStrokeSVG("none", "#123456", 2.5, "")
	a, err := Render(data, 200)
	if err != nil {
		t.Fatalf("render #1: %v", err)
	}
	b, err := Render(data, 200)
	if err != nil {
		t.Fatalf("render #2: %v", err)
	}
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Error("同一输入两次渲染结果不一致（textPath 缓冲复用未清空？）")
	}
}

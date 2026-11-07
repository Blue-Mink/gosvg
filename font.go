package gosvg

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

// —— 内置字体（零外部依赖回退） ——

var (
	fontOnce   sync.Once
	fontParsed *opentype.Font
)

// loadFont 惰性加载内置的 Go Regular 字体（无外部字体依赖）。
func loadFont() *opentype.Font {
	fontOnce.Do(func() {
		f, err := opentype.Parse(goregular.TTF)
		if err == nil {
			fontParsed = f
		}
	})
	return fontParsed
}

// —— 系统字体注册表（可选，由 AddFontDir 填充） ——

// fontRegistry 保存通过 AddFontDir 注册的系统字体，按 family 名索引。
// 内置 Go Regular 不在此表内，它是 resolveFont 未命中时的最终回退。
var fontRegistry = struct {
	sync.RWMutex
	families map[string]*fontFamily
}{families: map[string]*fontFamily{}}

// fontFamily 是一个 family 名下的多字重/字型面集合。
type fontFamily struct {
	faces map[string]*opentype.Font // key: "regular"/"bold"/"italic"/"bolditalic"
}

// faceKey 由 bold/italic 布尔生成稳定的字型键。
func faceKey(bold, italic bool) string {
	switch {
	case bold && italic:
		return "bolditalic"
	case bold:
		return "bold"
	case italic:
		return "italic"
	default:
		return "regular"
	}
}

// classifySubfamily 从字型 subfamily 名（"Bold"/"Book"/"Oblique"…）判断字重/字型。
func classifySubfamily(s string) (bold, italic bool) {
	ls := strings.ToLower(s)
	bold = strings.Contains(ls, "bold")
	italic = strings.Contains(ls, "italic") || strings.Contains(ls, "oblique")
	return
}

// normalizeFamily 规范化 family 名（小写、去首尾空白）用于索引与匹配。
func normalizeFamily(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// registerFont 把一个已解析字体按 (family, subfamily) 注册进注册表。
// 同名同键先注册者优先，不覆盖。
func registerFont(family, subfamily string, f *opentype.Font) {
	family = normalizeFamily(family)
	if family == "" || f == nil {
		return
	}
	bold, italic := classifySubfamily(subfamily)
	key := faceKey(bold, italic)

	fontRegistry.Lock()
	defer fontRegistry.Unlock()
	fam, ok := fontRegistry.families[family]
	if !ok {
		fam = &fontFamily{faces: map[string]*opentype.Font{}}
		fontRegistry.families[family] = fam
	}
	if _, exists := fam.faces[key]; !exists {
		fam.faces[key] = f
	}
}

// AddFontDir 递归扫描目录下的 .ttf/.otf 字体文件，按 family 名注册。
// 解析失败或不可读的字体静默跳过；重复调用增量合并。未命中字体目录时
// 直接返回 nil（本特性为可选，调用方可不依赖系统字体）。
func AddFontDir(dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".ttf" && ext != ".otf" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		f, err := opentype.Parse(data)
		if err != nil {
			return nil
		}
		family, err := f.Name(nil, sfnt.NameIDFamily)
		if err != nil {
			return nil
		}
		sub, _ := f.Name(nil, sfnt.NameIDSubfamily)
		registerFont(family, sub, f)
		return nil
	})
}

// bestFace 在该 family 下按 (bold, italic) 择优选一张字型，逐级回退。
func (fam *fontFamily) bestFace(bold, italic bool) *opentype.Font {
	if f, ok := fam.faces[faceKey(bold, italic)]; ok {
		return f
	}
	if f, ok := fam.faces[faceKey(bold, false)]; ok {
		return f
	}
	if f, ok := fam.faces[faceKey(false, italic)]; ok {
		return f
	}
	if f, ok := fam.faces[faceKey(false, false)]; ok {
		return f
	}
	for _, f := range fam.faces { // 兜底：该 family 下任意一张
		return f
	}
	return nil
}

// lookupFont 按 family 名精确查找（大小写不敏感），未命中返回 nil。
func lookupFont(family string) *opentype.Font {
	family = normalizeFamily(family)
	if family == "" {
		return nil
	}
	fontRegistry.RLock()
	defer fontRegistry.RUnlock()
	fam, ok := fontRegistry.families[family]
	if !ok {
		return nil
	}
	return fam.bestFace(false, false)
}

// lookupFace 按 family + 字重/字型查找，未命中回退到该 family 的 regular，仍无则 nil。
func lookupFace(family string, bold, italic bool) *opentype.Font {
	family = normalizeFamily(family)
	if family == "" {
		return nil
	}
	fontRegistry.RLock()
	defer fontRegistry.RUnlock()
	fam, ok := fontRegistry.families[family]
	if !ok {
		return nil
	}
	return fam.bestFace(bold, italic)
}

// firstFamilyMatching 返回注册表中第一个名字含 kw（且不含 exclude）的 family。
func firstFamilyMatching(kw, exclude string) string {
	fontRegistry.RLock()
	defer fontRegistry.RUnlock()
	for name := range fontRegistry.families {
		if exclude != "" && strings.Contains(name, exclude) {
			continue
		}
		if strings.Contains(name, kw) {
			return name
		}
	}
	return ""
}

// genericToConcrete 把 CSS 通用 family 映射到已注册的具体 family（若存在）。
func genericToConcrete(generic string) string {
	switch normalizeFamily(generic) {
	case "sans-serif", "system-ui", "ui-sans-serif":
		if lookupFont("dejavu sans") != nil {
			return "dejavu sans"
		}
		return firstFamilyMatching("sans", "mono")
	case "serif":
		if lookupFont("dejavu serif") != nil {
			return "dejavu serif"
		}
		return firstFamilyMatching("serif", "sans")
	case "monospace", "mono", "ui-monospace":
		if lookupFont("dejavu sans mono") != nil {
			return "dejavu sans mono"
		}
		return firstFamilyMatching("mono", "")
	}
	return ""
}

// splitFamilies 把 font-family 拆成候选列表（去引号、去空白、小写）。
func splitFamilies(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		p := strings.TrimSpace(part)
		p = strings.Trim(p, `"'`)
		if p == "" {
			continue
		}
		out = append(out, normalizeFamily(p))
	}
	return out
}

// resolveFont 按 font-family（含通用名与逗号列表）+ 字重/字型解析字体。
// 命中返回对应字型；未命中返回 nil（调用方回退内置 Go Regular）。
func resolveFont(family string, bold, italic bool) *opentype.Font {
	for _, fam := range splitFamilies(family) {
		if concrete := genericToConcrete(fam); concrete != "" {
			fam = concrete
		}
		if f := lookupFace(fam, bold, italic); f != nil {
			return f
		}
	}
	return nil
}

// resolveFace 解析字体（内置 Go Regular + 可选系统字体），未找到返回 nil。
// fontFamily 为空时直接返回内置字体；否则按 font-weight/font-style 择优。
func resolveFace(fontFamily, fontWeight, fontStyle string) *opentype.Font {
	f := loadFont()
	if fontFamily != "" {
		if ff := resolveFont(fontFamily, weightIsBold(fontWeight), isItalic(fontStyle)); ff != nil {
			f = ff
		}
	}
	return f
}

// weightIsBold 判断 font-weight 是否为粗体（bold/bolder/数值 ≥600）。
func weightIsBold(w string) bool {
	w = strings.ToLower(strings.TrimSpace(w))
	switch w {
	case "", "normal", "400", "lighter", "thin", "100", "200", "300", "500":
		return false
	case "bold", "bolder":
		return true
	}
	if n, err := strconv.Atoi(w); err == nil {
		return n >= 600
	}
	return false
}

// isItalic 判断 font-style 是否为斜体（italic/oblique）。
func isItalic(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "italic" || s == "oblique"
}

package gosvg

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
)

// DefaultImageDecoder 返回一个**仅解码 data: URI** 的安全默认解码器，
// 不触碰文件系统与网络。格式由标准库 image/png、image/jpeg、image/gif 提供，
// 不引入任何第三方依赖。
//
// 本函数只「返回」解码器，不会改写包级 ImageDecoder，因此默认渲染行为不变：
// 需要把它赋给 RenderOptions.ImageDecoder（推荐）或包级 ImageDecoder 才生效。
//
// 历史变更：v0.2.4 之前本函数对非 data: 的 href 会直接 os.Open(href) 读取本地
// 文件。服务端渲染不可信 SVG 时这构成任意本地文件读风险，故收紧为仅 data: URI。
// 仍需读本地文件的调用方请显式改用 LocalFileImageDecoder（并传白名单根目录）。
func DefaultImageDecoder() func(string) (image.Image, error) {
	return decodeDataURIOnly
}

// LocalFileImageDecoder 返回一个支持 data: URI **与本地文件路径**的解码器。
//
// 与 DefaultImageDecoder 的区别是显式的：调用本函数即表示调用方接受读取本地
// 文件的风险。allowedRoots 是本地路径白名单：
//
//   - 传入至少一个根目录：href 必须落在其中一个根目录内（含符号链接解析，
//     防 ../ 穿越与 symlink 逃逸），否则返回错误、不打开文件；
//   - 不传任何根目录：允许任意本地路径（等价于 v0.2.4 之前 DefaultImageDecoder
//     的行为），仅供完全可信的输入使用，处理不可信 SVG 时**不要**这样做。
//
// 远程 URL 从不被抓取（无 HTTP 客户端）；非 data:、非白名单内路径一律返回错误。
// 处理不可信 SVG 时，优先用 DefaultImageDecoder，或给本函数传入受限根目录。
func LocalFileImageDecoder(allowedRoots ...string) func(string) (image.Image, error) {
	roots := make([]string, 0, len(allowedRoots))
	for _, r := range allowedRoots {
		abs, err := filepath.Abs(r)
		if err != nil {
			continue
		}
		if resolved, err := filepath.EvalSymlinks(abs); err == nil {
			abs = resolved
		}
		roots = append(roots, abs)
	}
	return func(href string) (image.Image, error) {
		if strings.HasPrefix(href, "data:") {
			return decodeDataURI(href)
		}
		if strings.Contains(href, "://") {
			return nil, fmt.Errorf("gosvg: remote image URLs are not supported")
		}
		if !pathAllowed(href, roots) {
			return nil, fmt.Errorf("gosvg: image path %q is outside allowed roots", href)
		}
		f, err := os.Open(href)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		img, _, err := image.Decode(f)
		return img, err
	}
}

// pathAllowed 检查 href 是否落在白名单根目录内。roots 为空表示不限制（允许任意
// 本地路径）；非空时先解析绝对路径与符号链接，再用 Rel 判定是否越界。
func pathAllowed(href string, roots []string) bool {
	if len(roots) == 0 {
		return true
	}
	abs, err := filepath.Abs(href)
	if err != nil {
		return false
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	for _, root := range roots {
		rel, rerr := filepath.Rel(root, abs)
		if rerr != nil {
			continue
		}
		if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)) {
			return true
		}
	}
	return false
}

func decodeDataURI(s string) (image.Image, error) {
	comma := strings.IndexByte(s, ',')
	if comma < 0 {
		return nil, fmt.Errorf("gosvg: malformed data URI")
	}
	meta, payload := s[:comma], s[comma+1:]
	if !strings.Contains(meta, ";base64") {
		return nil, fmt.Errorf("gosvg: only base64 data URIs are supported")
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}

// decodeDataURIOnly 只解码 data: URI，不触碰文件系统与网络。
// 用于调用方未注入任何解码器时的兜底，使内联位图开箱可用。
func decodeDataURIOnly(href string) (image.Image, error) {
	if !strings.HasPrefix(href, "data:") {
		return nil, fmt.Errorf("gosvg: not a data URI")
	}
	return decodeDataURI(href)
}

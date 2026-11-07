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
	"strings"
)

// DefaultImageDecoder 返回一个零外部依赖的默认 <image> 解码器，支持：
//   - data:image/...;base64,<payload>（内联位图）
//   - 本地文件路径
//
// 格式由标准库 image/png、image/jpeg、image/gif 提供，不引入任何第三方依赖。
// 本函数只「返回」解码器，不会改写包级 ImageDecoder，因此默认渲染行为不变：
// 需要把它赋给 RenderOptions.ImageDecoder（推荐）或包级 ImageDecoder 才生效。
func DefaultImageDecoder() func(string) (image.Image, error) {
	return func(href string) (image.Image, error) {
		if strings.HasPrefix(href, "data:") {
			return decodeDataURI(href)
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

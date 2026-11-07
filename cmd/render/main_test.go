package main

import "testing"

func TestParseSize(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want int
	}{
		{"缺省", []string{"cmd", "in.svg", "out.png"}, 512},
		{"合法", []string{"cmd", "in.svg", "out.png", "128"}, 128},
		{"零", []string{"cmd", "in.svg", "out.png", "0"}, 512},
		{"负数", []string{"cmd", "in.svg", "out.png", "-3"}, 512},
		{"非数字", []string{"cmd", "in.svg", "out.png", "abc"}, 512},
	} {
		if got := parseSize(tc.args, 512); got != tc.want {
			t.Errorf("%s: parseSize() = %d, want %d", tc.name, got, tc.want)
		}
	}
}

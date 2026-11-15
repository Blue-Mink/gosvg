package gosvg

import "testing"

// SVG 1.1 §8.3.6/§8.3.7：S/T 的第一控制点仅在「上一段」确为 C/c/S/s
// （T 为 Q/q/T/t）时反射上一控制点；否则第一控制点与当前点重合。
//
// 曾因 parsePathData 把「当前指令」lastCmd 误当「上一段指令」用于反射判断，
// 导致 M 后紧跟的 leading-s / leading-t 把初值 (0,0) 的 lcx/lcy 拿来镜像，
// 曲线被甩歪。这里用「等价路径对比」锁定语义：不依赖 ToSVGPath 的具体格式，
// 只要求结果与「第一控制点 = 当前点」的显式 C/Q 逐段一致。
func assertSamePath(t *testing.T, got, want string) {
	t.Helper()
	g := parsePathData(got).ToSVGPath()
	w := parsePathData(want).ToSVGPath()
	if g != w {
		t.Errorf("parsePathData(%q).ToSVGPath() != parsePathData(%q).ToSVGPath()\n got = %s\nwant = %s",
			got, want, g, w)
	}
}

// M 后紧跟 s/S：上一段是 M，不应反射，第一控制点 = 当前点 (10,20)。
func TestParsePathLeadingSNoReflection(t *testing.T) {
	// 相对 s 30 40 50 60 → 第二控制点 (40,60)，终点 (60,80)。
	assertSamePath(t,
		"M 10 20 s 30 40 50 60",
		"M 10 20 C 10 20 40 60 60 80")
	// 大写 S（绝对坐标）同理。
	assertSamePath(t,
		"M 10 20 S 40 60 60 80",
		"M 10 20 C 10 20 40 60 60 80")
}

// M 后紧跟 t/T：同理，第一控制点 = 当前点。
func TestParsePathLeadingTNoReflection(t *testing.T) {
	assertSamePath(t,
		"M 10 20 t 50 60",
		"M 10 20 Q 10 20 60 80")
	assertSamePath(t,
		"M 10 20 T 60 80",
		"M 10 20 Q 10 20 60 80")
}

// 上一段是 C：S 必须反射（防止把反射整个关掉）。
func TestParsePathSReflectionAfterC(t *testing.T) {
	// 当前点 (60,80)，上一控制点 (40,60) → 反射第一控制点 (80,100)；
	// s 10 10 20 20 → 第二控制点 (70,90)，终点 (80,100)。
	assertSamePath(t,
		"M 10 20 C 15 25 40 60 60 80 s 10 10 20 20",
		"M 10 20 C 15 25 40 60 60 80 C 80 100 70 90 80 100")
}

// 上一段是 Q：T 必须反射。
func TestParsePathTReflectionAfterQ(t *testing.T) {
	assertSamePath(t,
		"M 10 20 Q 40 60 60 80 t 10 10",
		"M 10 20 Q 40 60 60 80 Q 80 100 70 90")
}

// 连续 S 段（显式 + 数字复用的隐式段）都应保持反射链。
func TestParsePathSReflectionChain(t *testing.T) {
	// 逐段展开：C 10 0 10 10 20 10；S 反射→C 30 10 30 20 40 20；
	// 再 S 反射→C 50 20 50 30 60 30。
	assertSamePath(t,
		"M 0 0 C 10 0 10 10 20 10 S 30 20 40 20 S 50 30 60 30",
		"M 0 0 C 10 0 10 10 20 10 C 30 10 30 20 40 20 C 50 20 50 30 60 30")
	// 数字复用：单个 S 指令带 8 参数 = 两段隐式 S，反射链同样要续上。
	assertSamePath(t,
		"M 0 0 C 10 0 10 10 20 10 S 30 20 40 20 50 30 60 30",
		"M 0 0 C 10 0 10 10 20 10 C 30 10 30 20 40 20 C 50 20 50 30 60 30")
}

// 非曲线指令（Z / A / Q 接 S、C 接 T）都应打断反射链。
func TestParsePathReflectionBreaks(t *testing.T) {
	// Z 之后的 S：上一段是 z，不反射。
	assertSamePath(t,
		"M 10 20 L 30 40 Z M 10 20 S 40 60 60 80",
		"M 10 20 L 30 40 Z M 10 20 C 10 20 40 60 60 80")
	// Q 之后的 S：S 只反射 C/S，不反射 Q。
	assertSamePath(t,
		"M 10 20 Q 40 60 60 80 S 90 100 110 120",
		"M 10 20 Q 40 60 60 80 C 60 80 90 100 110 120")
	// C 之后的 T：T 只反射 Q/T，不反射 C。
	assertSamePath(t,
		"M 10 20 C 15 25 40 60 60 80 T 110 120",
		"M 10 20 C 15 25 40 60 60 80 Q 60 80 110 120")
}

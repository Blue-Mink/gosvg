package rasterx

import (
	"math"
	"testing"

	"golang.org/x/image/math/fixed"
)

func mApproxEq(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestIdentityTransform(t *testing.T) {
	x, y := Identity.Transform(3, -4)
	if !mApproxEq(x, 3) || !mApproxEq(y, -4) {
		t.Fatalf("Identity.Transform(3,-4) = (%v,%v), want (3,-4)", x, y)
	}
}

func TestMultWithIdentityIsNoop(t *testing.T) {
	m := Identity.Translate(5, 7).Scale(2, 3)
	if got := m.Mult(Identity); got != m {
		t.Fatalf("m*Identity = %+v, want %+v", got, m)
	}
	if got := Identity.Mult(m); got != m {
		t.Fatalf("Identity*m = %+v, want %+v", got, m)
	}
}

func TestTranslateAffectsTransform(t *testing.T) {
	x, y := Identity.Translate(10, 20).Transform(1, 2)
	if !mApproxEq(x, 11) || !mApproxEq(y, 22) {
		t.Fatalf("got (%v,%v), want (11,22)", x, y)
	}
}

func TestInvertRoundTrip(t *testing.T) {
	m := Identity.Translate(3, -7).Scale(2.5, 1.25).Rotate(0.4)
	back := m.Mult(m.Invert())
	if !mApproxEq(back.A, 1) || !mApproxEq(back.B, 0) || !mApproxEq(back.C, 0) ||
		!mApproxEq(back.D, 1) || !mApproxEq(back.E, 0) || !mApproxEq(back.F, 0) {
		t.Fatalf("m*inv = %+v, want Identity", back)
	}
}

func TestTransformVectorAppliesLinearPart(t *testing.T) {
	x, y := Identity.TransformVector(3, -4)
	if !mApproxEq(x, 3) || !mApproxEq(y, -4) {
		t.Fatalf("Identity.TransformVector = (%v,%v), want (3,-4)", x, y)
	}
	sx, sy := Identity.Scale(2, 3).TransformVector(1, 1)
	if !mApproxEq(sx, 2) || !mApproxEq(sy, 3) {
		t.Fatalf("Scale(2,3).TransformVector(1,1) = (%v,%v), want (2,3)", sx, sy)
	}
}

func TestTFixedIdentity(t *testing.T) {
	p := fixed.Point26_6{X: fixed.I(1), Y: fixed.I(-2)}
	got := Identity.TFixed(p)
	if got.X != fixed.I(1) || got.Y != fixed.I(-2) {
		t.Fatalf("TFixed identity = (%v,%v), want (%v,%v)", got.X, got.Y, fixed.I(1), fixed.I(-2))
	}
}

func TestSkewIsLinear(t *testing.T) {
	m := Identity.SkewX(math.Pi / 4)
	x, y := m.Transform(0, 1)
	if !mApproxEq(y, 1) {
		t.Fatalf("SkewX should not change y, got %v", y)
	}
	if mApproxEq(x, 0) {
		t.Fatalf("SkewX(pi/4) should shift x for y=1, got %v", x)
	}
	m2 := Identity.SkewY(math.Pi / 4)
	x2, y2 := m2.Transform(1, 0)
	if !mApproxEq(x2, 1) {
		t.Fatalf("SkewY should not change x, got %v", x2)
	}
	if mApproxEq(y2, 0) {
		t.Fatalf("SkewY(pi/4) should shift y for x=1, got %v", y2)
	}
}

package tire

import (
	"math"
	"testing"
)

var testCoeffs = DemoCoefficients()

const testLoad = DemoVerticalLoad

func mustForce(t *testing.T, c Coefficients, load, slip float64) float64 {
	t.Helper()
	f, _, err := LongitudinalForce(c, load, slip)
	if err != nil {
		t.Fatalf("LongitudinalForce(%v) unexpected error: %v", slip, err)
	}
	return f
}

// 零滑移率必须产生零纵向力。
func TestZeroSlipZeroForce(t *testing.T) {
	f := mustForce(t, testCoeffs, testLoad, 0)
	if f != 0 {
		t.Fatalf("F(0) = %v, want exactly 0", f)
	}
}

// 滑移率取反，纵向力大小不变、方向相反（无水平偏移的奇对称性）。
func TestOddSymmetry(t *testing.T) {
	for _, x := range []float64{0.01, 0.05, 0.13, 0.4, 0.9, 1.5} {
		fp := mustForce(t, testCoeffs, testLoad, x)
		fn := mustForce(t, testCoeffs, testLoad, -x)
		if fp != -fn {
			t.Fatalf("F(%v)=%v, F(%v)=%v, want exact odd symmetry", x, fp, -x, fn)
		}
	}
}

// D 由 mu*Fz 换算时，载荷成倍放大，峰值力必须同比例放大。
func TestLoadScalingScalesPeakForce(t *testing.T) {
	r := ScanRange{Min: -1, Max: 1, Points: 2001}
	for _, k := range []float64{0.5, 1.7, 2.0, 3.3} {
		p1, _, err := FindPeak(testCoeffs, testLoad, r)
		if err != nil {
			t.Fatal(err)
		}
		p2, _, err := FindPeak(testCoeffs, k*testLoad, r)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(p2.Force-k*p1.Force) > 1e-9*math.Abs(k*p1.Force) {
			t.Fatalf("k=%v: peak force %v, want %v (scaled)", k, p2.Force, k*p1.Force)
		}
		// 峰值滑移率与载荷无关。
		if math.Abs(p2.Slip-p1.Slip) > 1e-9 {
			t.Fatalf("k=%v: peak slip moved from %v to %v", k, p1.Slip, p2.Slip)
		}
	}
}

// 标准形状下任意工况的 |F| 不得超过峰值幅度 D（留数值容差）。
func TestForceNeverExceedsPeakAmplitude(t *testing.T) {
	_, d, err := LongitudinalForce(testCoeffs, testLoad, 0)
	if err != nil {
		t.Fatal(err)
	}
	const tol = 1e-9
	for _, x := range Linspace(-3, 3, 2001) {
		f := mustForce(t, testCoeffs, testLoad, x)
		if math.Abs(f) > d*(1+tol) {
			t.Fatalf("|F(%v)| = %v exceeds D = %v", x, math.Abs(f), d)
		}
	}
}

// 扫描找到的峰值滑移率处，用单点公式复算必须与扫描报告的峰值力一致。
func TestSweepPeakConsistentWithPointEvaluation(t *testing.T) {
	ranges := []ScanRange{
		{Min: -1, Max: 1, Points: 2001},
		{Min: 0, Max: 0.6, Points: 501},
		{Min: -0.3, Max: 0.3, Points: 121},
	}
	for _, r := range ranges {
		peak, _, err := FindPeak(testCoeffs, testLoad, r)
		if err != nil {
			t.Fatal(err)
		}
		f := mustForce(t, testCoeffs, testLoad, peak.Slip)
		if f != peak.Force {
			t.Fatalf("range %+v: sweep reports %v, point re-evaluation gives %v", r, peak.Force, f)
		}
	}
}

// 扫描峰值不得小于网格上任意采样点的 |F|（在细化精度内）。
func TestSweepFindsGlobalMaxOnRange(t *testing.T) {
	r := ScanRange{Min: -1, Max: 1, Points: 2001}
	peak, _, err := FindPeak(testCoeffs, testLoad, r)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range Linspace(r.Min, r.Max, r.Points) {
		f := mustForce(t, testCoeffs, testLoad, x)
		if math.Abs(f) > math.Abs(peak.Force)+1e-9 {
			t.Fatalf("grid point F(%v)=%v beats reported peak %v", x, f, peak.Force)
		}
	}
}

// 载荷非正 → INVALID_LOAD；退化形状系数 → INVALID_SHAPE。
func TestValidation(t *testing.T) {
	cases := []struct {
		name string
		c    Coefficients
		load float64
		code string
	}{
		{"zero load", testCoeffs, 0, ErrCodeInvalidLoad},
		{"negative load", testCoeffs, -100, ErrCodeInvalidLoad},
		{"nan load", testCoeffs, math.NaN(), ErrCodeInvalidLoad},
		{"zero shape factor", Coefficients{B: 10, C: 0, E: 0.97, Mu: 1}, testLoad, ErrCodeInvalidShape},
		{"negative shape factor", Coefficients{B: 10, C: -1.9, E: 0.97, Mu: 1}, testLoad, ErrCodeInvalidShape},
		{"zero stiffness", Coefficients{B: 0, C: 1.9, E: 0.97, Mu: 1}, testLoad, ErrCodeInvalidShape},
		{"nan curvature", Coefficients{B: 10, C: 1.9, E: math.NaN(), Mu: 1}, testLoad, ErrCodeInvalidShape},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := LongitudinalForce(tc.c, tc.load, 0.1)
			verr, ok := err.(*Error)
			if !ok || verr.Code != tc.code {
				t.Fatalf("got %v, want code %s", err, tc.code)
			}
			if _, _, err := FindPeak(tc.c, tc.load, ScanRange{-1, 1, 101}); err == nil {
				t.Fatal("FindPeak must reject the same invalid input")
			}
		})
	}
}

// 峰值幅度换算：D 直接给定优先，否则 mu*Fz；两条路径必须同源。
func TestPeakAmplitudeResolution(t *testing.T) {
	d, err := PeakAmplitude(testCoeffs, testLoad)
	if err != nil {
		t.Fatal(err)
	}
	if d != testCoeffs.Mu*testLoad {
		t.Fatalf("D = %v, want mu*Fz = %v", d, testCoeffs.Mu*testLoad)
	}
	explicit := 3500.0
	c := testCoeffs
	c.D = &explicit
	d, err = PeakAmplitude(c, testLoad)
	if err != nil {
		t.Fatal(err)
	}
	if d != explicit {
		t.Fatalf("D = %v, want explicit %v", d, explicit)
	}
	// 单点与扫描必须使用同一个 D。
	if _, dPoint, err := LongitudinalForce(c, testLoad, 0.1); err != nil || dPoint != explicit {
		t.Fatalf("point path D = %v, err = %v", dPoint, err)
	}
	if _, dSweep, err := FindPeak(c, testLoad, ScanRange{-1, 1, 101}); err != nil || dSweep != explicit {
		t.Fatalf("sweep path D = %v, err = %v", dSweep, err)
	}
}

// 内置示范参数：小滑移率下力与滑移率同号，且大小低于峰值幅度。
func TestDemoCoefficientsSelfCheck(t *testing.T) {
	for _, x := range []float64{0.005, 0.01, 0.05} {
		f, d, err := LongitudinalForce(DemoCoefficients(), DemoVerticalLoad, x)
		if err != nil {
			t.Fatal(err)
		}
		if f <= 0 {
			t.Fatalf("F(%v) = %v, want positive (same direction as slip)", x, f)
		}
		if math.Abs(f) >= d {
			t.Fatalf("F(%v) = %v, want |F| < D = %v", x, f, d)
		}
	}
}

// 滑移率越界只告警、不报错。
func TestSlipOutOfRangeWarnsOnly(t *testing.T) {
	if w := SlipWarnings(1.5); len(w) == 0 {
		t.Fatal("expect warning for |slip| > 1")
	}
	if w := SlipWarnings(0.5); w != nil {
		t.Fatalf("no warning expected, got %v", w)
	}
	if _, _, err := LongitudinalForce(testCoeffs, testLoad, 1.5); err != nil {
		t.Fatalf("out-of-range slip must still compute, got %v", err)
	}
}

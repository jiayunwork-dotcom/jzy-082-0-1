package tire

import "math"

// ScanRange 描述一次滑移区间扫描的范围与分辨率。
type ScanRange struct {
	Min    float64 // 滑移率下限
	Max    float64 // 滑移率上限
	Points int     // 粗扫描采样点数（>= 2）
}

// Peak 是扫描寻峰的结果：曲线上纵向力绝对值最大的工况点。
type Peak struct {
	Slip  float64 // 峰值滑移率
	Force float64 // 峰值力（用单点公式在 Slip 处求值得到）
}

// FindPeak 在 [Min, Max] 上扫描 |F| 的最大值：先均匀粗扫定位，
// 再在最优点所在邻域内做黄金分割细化。
//
// 峰值幅度 D 由 PeakAmplitude 解析一次并贯穿整个扫描；
// 返回的 Force 用与单点计算完全相同的 evalWithAmplitude 在
// 最终峰值滑移率处求值，因此「扫描峰值点用单点公式复算」必然一致。
//
// 扫描参数非法时返回错误，绝不返回不完整的结果。
func FindPeak(c Coefficients, verticalLoad float64, r ScanRange) (Peak, float64, error) {
	if err := Validate(c, verticalLoad); err != nil {
		return Peak{}, 0, err
	}
	d, err := PeakAmplitude(c, verticalLoad)
	if err != nil {
		return Peak{}, 0, err
	}
	if math.IsNaN(r.Min) || math.IsNaN(r.Max) || r.Min >= r.Max {
		return Peak{}, 0, &Error{Code: ErrCodeInvalidShape, Message: "scan range must satisfy min < max"}
	}
	if r.Points < 2 {
		return Peak{}, 0, &Error{Code: ErrCodeInvalidShape, Message: "scan points must be >= 2"}
	}

	absF := func(x float64) float64 { return math.Abs(evalWithAmplitude(c, d, x)) }

	// 粗扫描：完整扫完整个区间后才允许进入细化阶段。
	step := (r.Max - r.Min) / float64(r.Points-1)
	best := 0
	bestVal := math.Inf(-1)
	for i := 0; i < r.Points; i++ {
		v := absF(r.Min + float64(i)*step)
		if v > bestVal {
			bestVal, best = v, i
		}
	}

	// 以最优点左右邻点为 bracket，黄金分割细化 |F| 的极大值。
	lo := r.Min + float64(max(best-1, 0))*step
	hi := r.Min + float64(min(best+1, r.Points-1))*step
	peakSlip := goldenMax(absF, lo, hi)

	// 曲线关于原点奇对称，|F| 的极大值成对出现在 ±x 处。
	// 归一化到非负滑移率一侧，保证结果确定（力用同一公式在该点求值）。
	if peakSlip < 0 {
		peakSlip = -peakSlip
	}

	return Peak{Slip: peakSlip, Force: evalWithAmplitude(c, d, peakSlip)}, d, nil
}

// SampleCurve 在给定滑移率网格上逐点求值，返回整条曲线的采样点。
// 与 FindPeak 使用同一个 D 来源和同一个求值函数。
func SampleCurve(c Coefficients, verticalLoad float64, slips []float64) (points []Peak, peakAmplitude float64, err error) {
	if err := Validate(c, verticalLoad); err != nil {
		return nil, 0, err
	}
	d, err := PeakAmplitude(c, verticalLoad)
	if err != nil {
		return nil, 0, err
	}
	points = make([]Peak, len(slips))
	for i, s := range slips {
		points[i] = Peak{Slip: s, Force: evalWithAmplitude(c, d, s)}
	}
	return points, d, nil
}

// Linspace 生成 [min, max] 上 n 个等距点。n < 2 时返回 nil。
func Linspace(min, max float64, n int) []float64 {
	if n < 2 {
		return nil
	}
	out := make([]float64, n)
	step := (max - min) / float64(n-1)
	for i := range out {
		out[i] = min + float64(i)*step
	}
	return out
}

// goldenMax 在 [a, b] 上用黄金分割搜索 f 的极大值点。
func goldenMax(f func(float64) float64, a, b float64) float64 {
	const (
		gr        = 0.6180339887498949 // (sqrt(5)-1)/2
		tolerance = 1e-12
	)
	c := b - gr*(b-a)
	d := a + gr*(b-a)
	for math.Abs(b-a) > tolerance {
		if f(c) < f(d) {
			a = c
			c = d
			d = a + gr*(b-a)
		} else {
			b = d
			d = c
			c = b - gr*(b-a)
		}
	}
	return (a + b) / 2
}

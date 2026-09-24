package tire

import "math"

// LongitudinalForce 计算给定垂直载荷与纵向滑移率下的轮胎纵向力。
//
// 公式（纯纵向、无偏移的 Magic Formula）：
//
//	F = D * sin(C * atan(phi))，其中 phi 由 CorrectedArgument 给出。
//
// 峰值幅度 D 一律经 PeakAmplitude 解析，与扫描寻峰共用同一来源。
func LongitudinalForce(c Coefficients, verticalLoad, slip float64) (force float64, peakAmplitude float64, err error) {
	if err := Validate(c, verticalLoad); err != nil {
		return 0, 0, err
	}
	d, err := PeakAmplitude(c, verticalLoad)
	if err != nil {
		return 0, 0, err
	}
	return evalWithAmplitude(c, d, slip), d, nil
}

// evalWithAmplitude 是主曲线求值的唯一实现。单点计算与扫描寻峰
// 都必须走这里，保证同一组系数下公式只有一份。
func evalWithAmplitude(c Coefficients, d, slip float64) float64 {
	return d * math.Sin(c.C*math.Atan(CorrectedArgument(slip, c.B, c.E)))
}

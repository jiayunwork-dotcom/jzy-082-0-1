package tire

// PeakAmplitude 解析本组系数下实际使用的峰值幅度 D。
//
// 这是「D 要么直接给定、要么由 摩擦系数 × 垂直载荷 得到」
// 这条换算关系在代码库中的唯一定义点：单点计算（LongitudinalForce）
// 与扫描寻峰（FindPeak）都调用它，禁止在别处重复实现。
func PeakAmplitude(c Coefficients, verticalLoad float64) (float64, error) {
	if c.D != nil {
		if *c.D <= 0 {
			return 0, &Error{Code: ErrCodeInvalidShape, Message: "peak amplitude D must be positive"}
		}
		return *c.D, nil
	}
	d := c.Mu * verticalLoad
	if d <= 0 {
		return 0, &Error{Code: ErrCodeInvalidShape, Message: "friction coefficient mu must be positive when D is not given"}
	}
	return d, nil
}

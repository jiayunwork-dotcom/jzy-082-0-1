package tire

import "math"

// CorrectedArgument 对纵向滑移率做刚度（B）与曲率（E）修正，
// 得到送进主曲线反正切的中间变量：
//
//	phi = B*x - E*(B*x - atan(B*x))
//
// 模型不含水平偏移，故 x = 0 时 phi = 0，保证零滑移得零力、
// 滑移变号时修正量也随之变号（奇对称）。
func CorrectedArgument(slip, b, e float64) float64 {
	bx := b * slip
	return bx - e*(bx-math.Atan(bx))
}

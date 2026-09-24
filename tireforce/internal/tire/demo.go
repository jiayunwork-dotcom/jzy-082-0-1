package tire

// DemoVerticalLoad 是内置示范工况的垂直载荷（乘用车单轮量级，牛顿）。
const DemoVerticalLoad = 4000.0

// DemoCoefficients 返回一份乘用车量级的示范形状系数。
//
// 该参数下小滑移率时纵向力方向与滑移率一致、大小低于峰值幅度，
// 可用于加载后快速自检。每次调用返回新的副本，调用方互不影响。
func DemoCoefficients() Coefficients {
	return Coefficients{
		B:  10.0,
		C:  1.9,
		E:  0.97,
		Mu: 1.0, // D 由 Mu * 垂直载荷 换算
	}
}

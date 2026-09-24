// Package tire 实现纯纵向经验轮胎模型（Magic Formula，无水平/垂直偏移）。
//
// 包内按职责拆分：
//   - slip.go      滑移率的刚度/曲率修正
//   - formula.go   主曲线公式求值
//   - amplitude.go 峰值幅度 D 与垂直载荷的换算（唯一定义点）
//   - sweep.go     滑移区间扫描寻峰
//   - validate.go  参数合法性校验
//   - params.go    系数类型与内置示范参数
package tire

// Coefficients 描述一条纯纵向力—滑移曲线的全部形状参数。
//
// 峰值幅度 D 为可选：直接给定（D != nil）时优先使用；
// 否则由 Mu * 垂直载荷 在 PeakAmplitude 中换算得到。
type Coefficients struct {
	B  float64  // 刚度因子（stiffness factor）
	C  float64  // 形状因子（shape factor）
	E  float64  // 曲率因子（curvature factor）
	D  *float64 // 峰值幅度（peak value），可选
	Mu float64  // 摩擦系数，D 未直接给定时参与换算
}

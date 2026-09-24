package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"tireforce/internal/tire"
)

// 默认扫描设置：覆盖常规滑移率全程，分辨率足够定位峰值。
const (
	defaultScanMin    = -1.0
	defaultScanMax    = 1.0
	defaultScanPoints = 2001
)

// handleForce 计算单点纵向力：载荷 + 滑移率 + 系数 → 纵向力与实际使用的峰值幅度。
func handleForce(c *gin.Context) {
	var req forceRequest
	if !bindJSON(c, &req) {
		return
	}
	coeff, load, ok := parseCommon(c, req.Coefficients, req.VerticalLoad)
	slip := req.Slip
	if !ok {
		return
	}
	if slip == nil {
		fail(c, http.StatusBadRequest, "BAD_REQUEST", "slip is required")
		return
	}

	force, d, err := tire.LongitudinalForce(coeff, load, *slip)
	if !respondDomainError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"force":          force,
		"peak_amplitude": d,
		"warnings":       tire.SlipWarnings(*slip),
	})
}

// handlePeak 在滑移区间上扫描寻峰：系数 + 载荷 → 峰值滑移率与峰值力。
func handlePeak(c *gin.Context) {
	var req peakRequest
	if !bindJSON(c, &req) {
		return
	}
	coeff, load, ok := parseCommon(c, req.Coefficients, req.VerticalLoad)
	if !ok {
		return
	}

	r := tire.ScanRange{Min: defaultScanMin, Max: defaultScanMax, Points: defaultScanPoints}
	if req.Scan != nil {
		if req.Scan.Min != nil {
			r.Min = *req.Scan.Min
		}
		if req.Scan.Max != nil {
			r.Max = *req.Scan.Max
		}
		if req.Scan.Points != nil {
			r.Points = *req.Scan.Points
		}
	}

	peak, d, err := tire.FindPeak(coeff, load, r)
	if !respondDomainError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"peak_slip":      peak.Slip,
		"peak_force":     peak.Force,
		"peak_amplitude": d,
	})
}

// handleCurve 在滑移率网格上采样整条力—滑移曲线。
func handleCurve(c *gin.Context) {
	var req curveRequest
	if !bindJSON(c, &req) {
		return
	}
	coeff, load, ok := parseCommon(c, req.Coefficients, req.VerticalLoad)
	if !ok {
		return
	}

	var slips []float64
	switch {
	case len(req.Slips) > 0 && req.Grid != nil:
		fail(c, http.StatusBadRequest, "BAD_REQUEST", "provide either slips or grid, not both")
		return
	case len(req.Slips) > 0:
		slips = req.Slips
	case req.Grid != nil:
		if req.Grid.Min == nil || req.Grid.Max == nil || req.Grid.Points == nil {
			fail(c, http.StatusBadRequest, "BAD_REQUEST", "grid requires min, max and points")
			return
		}
		slips = tire.Linspace(*req.Grid.Min, *req.Grid.Max, *req.Grid.Points)
		if slips == nil {
			fail(c, http.StatusBadRequest, "BAD_REQUEST", "grid points must be >= 2")
			return
		}
	default:
		fail(c, http.StatusBadRequest, "BAD_REQUEST", "either slips or grid is required")
		return
	}

	points, d, err := tire.SampleCurve(coeff, load, slips)
	if !respondDomainError(c, err) {
		return
	}
	out := make([]pointResponse, len(points))
	for i, p := range points {
		out[i] = pointResponse{Slip: p.Slip, Force: p.Force}
	}
	c.JSON(http.StatusOK, gin.H{
		"points":         out,
		"peak_amplitude": d,
	})
}

// handleDemo 返回内置乘用车示范参数及一个小滑移率自检采样。
func handleDemo(c *gin.Context) {
	coeff := tire.DemoCoefficients()
	const sampleSlip = 0.05
	force, d, err := tire.LongitudinalForce(coeff, tire.DemoVerticalLoad, sampleSlip)
	if !respondDomainError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"coefficients":  gin.H{"B": coeff.B, "C": coeff.C, "E": coeff.E, "mu": coeff.Mu},
		"vertical_load": tire.DemoVerticalLoad,
		"sample": gin.H{
			"slip":           sampleSlip,
			"force":          force,
			"peak_amplitude": d,
		},
	})
}

// parseCommon 解析并校验各入口共有的载荷与系数字段。
func parseCommon(c *gin.Context, cr *coefficientsRequest, load *float64) (tire.Coefficients, float64, bool) {
	coeff, err := cr.toDomain()
	if err != nil {
		fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return tire.Coefficients{}, 0, false
	}
	if load == nil {
		fail(c, http.StatusBadRequest, "BAD_REQUEST", "vertical_load is required")
		return tire.Coefficients{}, 0, false
	}
	return coeff, *load, true
}

func bindJSON(c *gin.Context, v any) bool {
	if err := c.ShouldBindJSON(v); err != nil {
		fail(c, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON body: "+err.Error())
		return false
	}
	return true
}

// respondDomainError 把内核层的结构化错误映射为 HTTP 400；
// err 为 nil 时返回 true 表示继续，否则已写出响应、返回 false。
func respondDomainError(c *gin.Context, err error) bool {
	if err == nil {
		return true
	}
	var verr *tire.Error
	if errors.As(err, &verr) {
		fail(c, http.StatusBadRequest, verr.Code, verr.Message)
		return false
	}
	fail(c, http.StatusInternalServerError, "INTERNAL", err.Error())
	return false
}

func fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, newErrorBody(code, message))
}

package api

import (
	"fmt"

	"tireforce/internal/tire"
)

// coefficientsRequest 是系数在 JSON 中的表示。
// 各字段用指针以区分「未提供」与「零值」，便于在计算前完成校验。
type coefficientsRequest struct {
	B  *float64 `json:"B"`
	C  *float64 `json:"C"`
	E  *float64 `json:"E"`
	D  *float64 `json:"D"`  // 可选：峰值幅度，直接给定
	Mu *float64 `json:"mu"` // 可选：摩擦系数，D 缺省时参与换算
}

func (r *coefficientsRequest) toDomain() (tire.Coefficients, error) {
	if r == nil || r.B == nil || r.C == nil || r.E == nil {
		return tire.Coefficients{}, fmt.Errorf("coefficients must include B, C and E")
	}
	c := tire.Coefficients{B: *r.B, C: *r.C, E: *r.E, D: r.D}
	if r.Mu != nil {
		c.Mu = *r.Mu
	}
	return c, nil
}

type forceRequest struct {
	VerticalLoad *float64             `json:"vertical_load"`
	Slip         *float64             `json:"slip"`
	Coefficients *coefficientsRequest `json:"coefficients"`
}

type scanRequest struct {
	Min    *float64 `json:"min"`
	Max    *float64 `json:"max"`
	Points *int     `json:"points"`
}

type peakRequest struct {
	VerticalLoad *float64             `json:"vertical_load"`
	Coefficients *coefficientsRequest `json:"coefficients"`
	Scan         *scanRequest         `json:"scan"` // 可选，缺省 [-1, 1] × 2001 点
}

type gridRequest struct {
	Min    *float64 `json:"min"`
	Max    *float64 `json:"max"`
	Points *int     `json:"points"`
}

type curveRequest struct {
	VerticalLoad *float64             `json:"vertical_load"`
	Coefficients *coefficientsRequest `json:"coefficients"`
	Slips        []float64            `json:"slips"` // 显式滑移率网格，与 grid 二选一
	Grid         *gridRequest         `json:"grid"`  // 等距网格，与 slips 二选一
}

type pointResponse struct {
	Slip  float64 `json:"slip"`
	Force float64 `json:"force"`
}

type peakResponse struct {
	Slip  float64 `json:"slip"`
	Force float64 `json:"force"`
}

// errorBody 是所有非法输入的统一结构化错误响应。
type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func newErrorBody(code, message string) errorBody {
	var b errorBody
	b.Error.Code = code
	b.Error.Message = message
	return b
}

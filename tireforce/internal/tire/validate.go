package tire

import "math"

// 校验错误码，接口层据此生成结构化错误响应。
const (
	ErrCodeInvalidLoad  = "INVALID_LOAD"
	ErrCodeInvalidShape = "INVALID_SHAPE"
)

// Error 是参数非法时返回的结构化错误。
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

// Validate 在任何计算之前检查载荷与形状系数的合法性。
//
//   - 垂直载荷非正（含 NaN）→ INVALID_LOAD
//   - 形状因子 C 非正、刚度因子 B 为零（曲线不存在有意义的峰值），
//     或任一系数为 NaN → INVALID_SHAPE
func Validate(c Coefficients, verticalLoad float64) error {
	if math.IsNaN(verticalLoad) || verticalLoad <= 0 {
		return &Error{Code: ErrCodeInvalidLoad, Message: "vertical load must be positive"}
	}
	if math.IsNaN(c.B) || math.IsNaN(c.C) || math.IsNaN(c.E) || math.IsNaN(c.Mu) {
		return &Error{Code: ErrCodeInvalidShape, Message: "coefficients must not be NaN"}
	}
	if c.C <= 0 {
		return &Error{Code: ErrCodeInvalidShape, Message: "shape factor C must be positive"}
	}
	if c.B == 0 {
		return &Error{Code: ErrCodeInvalidShape, Message: "stiffness factor B must be non-zero"}
	}
	return nil
}

// SlipWarningLimit 是常规纵向滑移率的绝对值上限（|x| = 1 即纯滑动）。
// 超出该范围不视为错误，但接口层会附带告警。
const SlipWarningLimit = 1.0

// SlipWarnings 返回滑移率越界告警；未越界时返回 nil。
func SlipWarnings(slip float64) []string {
	if math.Abs(slip) > SlipWarningLimit {
		return []string{"SLIP_OUT_OF_RANGE: |slip| exceeds 1.0, result computed anyway"}
	}
	return nil
}

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func doJSON(h http.Handler, method, path string, body any) (int, map[string]any, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return 0, nil, err
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var out map[string]any
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			return rec.Code, nil, fmt.Errorf("response not JSON: %w", err)
		}
	}
	return rec.Code, out, nil
}

func mustDoJSON(t *testing.T, h http.Handler, method, path string, body any) (int, map[string]any) {
	t.Helper()
	code, out, err := doJSON(h, method, path, body)
	if err != nil {
		t.Fatal(err)
	}
	return code, out
}

func demoBody(slip float64) map[string]any {
	return map[string]any{
		"vertical_load": 4000,
		"slip":          slip,
		"coefficients":  map[string]any{"B": 10, "C": 1.9, "E": 0.97, "mu": 1.0},
	}
}

func TestForceEndpoint(t *testing.T) {
	h := NewRouter()
	code, resp := mustDoJSON(t, h, http.MethodPost, "/v1/force", demoBody(0.05))
	if code != http.StatusOK {
		t.Fatalf("status %d, resp %v", code, resp)
	}
	if resp["force"].(float64) <= 0 {
		t.Fatalf("force should be positive for positive slip, got %v", resp["force"])
	}
	if resp["peak_amplitude"].(float64) != 4000.0 {
		t.Fatalf("peak_amplitude = %v, want 4000", resp["peak_amplitude"])
	}
}

func TestForceEndpointOddSymmetryOverHTTP(t *testing.T) {
	h := NewRouter()
	_, pos := mustDoJSON(t, h, http.MethodPost, "/v1/force", demoBody(0.1))
	_, neg := mustDoJSON(t, h, http.MethodPost, "/v1/force", demoBody(-0.1))
	if pos["force"].(float64) != -neg["force"].(float64) {
		t.Fatalf("F(0.1)=%v, F(-0.1)=%v", pos["force"], neg["force"])
	}
}

func TestPeakEndpointConsistentWithForceEndpoint(t *testing.T) {
	h := NewRouter()
	body := map[string]any{
		"vertical_load": 4000,
		"coefficients":  map[string]any{"B": 10, "C": 1.9, "E": 0.97, "mu": 1.0},
	}
	code, peak := mustDoJSON(t, h, http.MethodPost, "/v1/peak", body)
	if code != http.StatusOK {
		t.Fatalf("status %d, resp %v", code, peak)
	}
	// 用单点入口在峰值滑移率处复算，必须与扫描结果一致。
	recheck := demoBody(peak["peak_slip"].(float64))
	_, point := mustDoJSON(t, h, http.MethodPost, "/v1/force", recheck)
	if point["force"].(float64) != peak["peak_force"].(float64) {
		t.Fatalf("peak_force %v != point re-eval %v", peak["peak_force"], point["force"])
	}
}

func TestCurveEndpoint(t *testing.T) {
	h := NewRouter()
	body := map[string]any{
		"vertical_load": 4000,
		"coefficients":  map[string]any{"B": 10, "C": 1.9, "E": 0.97, "mu": 1.0},
		"grid":          map[string]any{"min": -0.5, "max": 0.5, "points": 11},
	}
	code, resp := mustDoJSON(t, h, http.MethodPost, "/v1/curve", body)
	if code != http.StatusOK {
		t.Fatalf("status %d, resp %v", code, resp)
	}
	pts := resp["points"].([]any)
	if len(pts) != 11 {
		t.Fatalf("got %d points, want 11", len(pts))
	}
	// 中点（滑移率 0）力必须为零。
	mid := pts[5].(map[string]any)
	if mid["slip"].(float64) != 0 || mid["force"].(float64) != 0 {
		t.Fatalf("mid point = %v, want zero slip and zero force", mid)
	}
}

func TestValidationErrorsAreStructured(t *testing.T) {
	h := NewRouter()
	cases := []struct {
		name string
		body map[string]any
		code string
	}{
		{"bad load", map[string]any{"vertical_load": -1, "slip": 0.1, "coefficients": map[string]any{"B": 10, "C": 1.9, "E": 0.97, "mu": 1.0}}, "INVALID_LOAD"},
		{"bad shape", map[string]any{"vertical_load": 4000, "slip": 0.1, "coefficients": map[string]any{"B": 10, "C": 0, "E": 0.97, "mu": 1.0}}, "INVALID_SHAPE"},
		{"zero stiffness", map[string]any{"vertical_load": 4000, "slip": 0.1, "coefficients": map[string]any{"B": 0, "C": 1.9, "E": 0.97, "mu": 1.0}}, "INVALID_SHAPE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, resp := mustDoJSON(t, h, http.MethodPost, "/v1/force", tc.body)
			if status != http.StatusBadRequest {
				t.Fatalf("status %d, want 400", status)
			}
			e := resp["error"].(map[string]any)
			if e["code"] != tc.code || e["message"] == "" {
				t.Fatalf("error body = %v, want code %s", e, tc.code)
			}
		})
	}
}

func TestSlipWarningPresentButComputed(t *testing.T) {
	h := NewRouter()
	code, resp := mustDoJSON(t, h, http.MethodPost, "/v1/force", demoBody(1.5))
	if code != http.StatusOK {
		t.Fatalf("status %d, want 200 (compute anyway)", code)
	}
	w := resp["warnings"].([]any)
	if len(w) == 0 {
		t.Fatal("expect slip out-of-range warning")
	}
}

func TestDemoEndpoint(t *testing.T) {
	h := NewRouter()
	code, resp := mustDoJSON(t, h, http.MethodGet, "/v1/demo", nil)
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	sample := resp["sample"].(map[string]any)
	force := sample["force"].(float64)
	if force <= 0 || math.Abs(force) >= sample["peak_amplitude"].(float64) {
		t.Fatalf("demo self-check failed: %v", sample)
	}
}

// 并发请求：不同系数、不同载荷的请求并行打进来，结果必须与串行计算逐一相同。
func TestConcurrentRequestsIsolated(t *testing.T) {
	h := NewRouter()
	type job struct {
		b, c, e, mu, load, slip float64
	}
	jobs := make([]job, 64)
	for i := range jobs {
		jobs[i] = job{
			b: 8 + float64(i%5), c: 1.6 + 0.1*float64(i%4), e: 0.9 + 0.02*float64(i%3),
			mu: 0.8 + 0.05*float64(i%6), load: 2500 + 100*float64(i), slip: -0.4 + 0.0125*float64(i),
		}
	}
	expected := make([]float64, len(jobs))
	for i, j := range jobs {
		status, resp, err := doJSON(h, http.MethodPost, "/v1/force", map[string]any{
			"vertical_load": j.load, "slip": j.slip,
			"coefficients": map[string]any{"B": j.b, "C": j.c, "E": j.e, "mu": j.mu},
		})
		if err != nil || status != http.StatusOK {
			t.Fatalf("setup request failed: %d %v", status, err)
		}
		expected[i] = resp["force"].(float64)
	}

	var wg sync.WaitGroup
	errs := make(chan string, len(jobs)*4)
	for round := 0; round < 4; round++ {
		for i, j := range jobs {
			wg.Add(1)
			go func(i int, j job) {
				defer wg.Done()
				status, resp, err := doJSON(h, http.MethodPost, "/v1/force", map[string]any{
					"vertical_load": j.load, "slip": j.slip,
					"coefficients": map[string]any{"B": j.b, "C": j.c, "E": j.e, "mu": j.mu},
				})
				if err != nil || status != http.StatusOK {
					errs <- fmt.Sprintf("job %d: status %d err %v", i, status, err)
					return
				}
				if resp["force"].(float64) != expected[i] {
					errs <- fmt.Sprintf("job %d: force %v != expected %v", i, resp["force"], expected[i])
				}
			}(i, j)
		}
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
}

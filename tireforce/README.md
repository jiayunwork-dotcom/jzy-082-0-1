# tireforce — 轮胎纵向力计算服务

纯纵向经验轮胎模型（Magic Formula，无水平/垂直偏移）的 HTTP 微服务，
供整车纵向牵引/制动仿真主循环高频调用。Go 1.22 + Gin，只做计算，无网页。

## 模型

```
F = D · sin(C · atan(φ))
φ = B·x − E·(B·x − atan(B·x))
```

- `x`：纵向滑移率；`F`：纵向力（N）
- `B` 刚度因子、`C` 形状因子、`E` 曲率因子、`D` 峰值幅度
- `D` 要么直接给定，要么由 `mu × 垂直载荷` 换算——该换算只定义在
  `internal/tire/amplitude.go` 的 `PeakAmplitude`，单点与扫描共用

## 构建与运行（容器）

```bash
docker build -t tireforce .
docker run -p 8080:8080 tireforce
```

本地开发：`go test ./... && go run ./cmd/server`（端口用 `PORT` 覆盖，默认 8080）。

## 接口

### POST /v1/force — 单点纵向力

```json
{"vertical_load": 4000, "slip": 0.05,
 "coefficients": {"B": 10, "C": 1.9, "E": 0.97, "mu": 1.0}}
```
→ `{"force": 2793.5…, "peak_amplitude": 4000, "warnings": null}`

`coefficients` 中给 `"D": 3800` 则直接使用该峰值幅度，忽略 `mu`。
`|slip| > 1` 时 `warnings` 带 `SLIP_OUT_OF_RANGE`，仍正常计算。

### POST /v1/peak — 扫描寻峰

```json
{"vertical_load": 4000,
 "coefficients": {"B": 10, "C": 1.9, "E": 0.97, "mu": 1.0},
 "scan": {"min": -1, "max": 1, "points": 2001}}
```
→ `{"peak_slip": 0.0772…, "peak_force": 3999.9…, "peak_amplitude": 4000}`

`scan` 可省略（默认 [-1, 1] × 2001 点）。粗扫 + 黄金分割细化；
返回的 `peak_force` 用与单点完全相同的公式在 `peak_slip` 处求值，
二者必然一致。扫描参数非法时返回错误，绝不返回不完整结果。

### POST /v1/curve — 曲线采样

```json
{"vertical_load": 4000, "coefficients": {"B": 10, "C": 1.9, "E": 0.97, "mu": 1.0},
 "grid": {"min": -0.5, "max": 0.5, "points": 101}}
```
或用 `"slips": [-0.2, -0.1, 0, 0.1]` 显式给网格（二选一）。
→ `{"points": [{"slip": …, "force": …}, …], "peak_amplitude": 4000}`

### GET /v1/demo — 内置乘用车示范参数自检

返回示范系数（B=10, C=1.9, E=0.97, mu=1.0, Fz=4000 N）及小滑移率采样：
力与滑移率同向、大小低于峰值幅度。

### GET /healthz — 健康检查

## 错误

非法输入一律 HTTP 400 + 结构化错误：

```json
{"error": {"code": "INVALID_LOAD",  "message": "vertical load must be positive"}}
{"error": {"code": "INVALID_SHAPE", "message": "shape factor C must be positive"}}
```

- `INVALID_LOAD`：垂直载荷非正（含 NaN）
- `INVALID_SHAPE`：形状因子 C 非正、刚度因子 B 为零、系数 NaN、峰值幅度非正

## 代码结构

```
cmd/server/            进程入口
internal/tire/         计算内核（无状态，并发安全）
  slip.go              滑移率刚度/曲率修正
  formula.go           主曲线求值（唯一实现 evalWithAmplitude）
  amplitude.go         峰值幅度 D 换算（唯一定义点）
  sweep.go             滑移区间扫描寻峰 + 曲线采样
  validate.go          参数合法性校验
  params.go, demo.go   系数类型与内置示范参数
internal/api/          HTTP 接口层（Gin）
```

## 测试锁死的性质

- 零滑移 → 零力
- 奇对称：F(−x) = −F(x)
- D 由 mu·Fz 换算时，载荷放大 k 倍 → 峰值力放大 k 倍
- 任意工况 |F| ≤ D（数值容差内）
- 扫描峰值点用单点公式复算，结果与扫描报告完全一致
- 并发请求之间系数与中间量互不污染

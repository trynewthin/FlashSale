package model

import (
	"encoding/json"
	"time"
)

var (
	timeNowUnixMilli = func() int64 { return time.Now().UnixMilli() }
	timeParse        = time.Parse
	timeUnixMilli    = func(ms int64) time.Time { return time.UnixMilli(ms) }
)

// PerfProgress 是 seckillload 每秒输出的 progress JSON 解析结果。
// 后端在 pipeToLog 中拦截并解析，通过 SSE perf_progress 事件实时推送给前端。
type PerfProgress struct {
	Timestamp                  int64    `json:"timestamp"`
	Label                      string   `json:"label"`
	QPS                        float64  `json:"qps"`
	P95LatencyMs               float64  `json:"p95LatencyMs"`
	SuccessRate                float64  `json:"successRate"`
	RejectRate                 float64  `json:"rejectRate"`
	SystemErrorRate            float64  `json:"systemErrorRate"`
	NetworkErrorRate           float64  `json:"networkErrorRate"`
	StockDeductRate            float64  `json:"stockDeductionRate"`
	PromQps                    *float64 `json:"promQps,omitempty"`
	PromP99LatencyMs           *float64 `json:"promP99LatencyMs,omitempty"`
	PromErrorRate              *float64 `json:"promErrorRate,omitempty"`
	PurchaseKafkaPublishRate   *float64 `json:"purchaseKafkaPublishRate,omitempty"`
	PurchaseKafkaPublishFailed *float64 `json:"purchaseKafkaPublishFailed,omitempty"`
	OrderStateConsumeRate      *float64 `json:"orderStateConsumeRate,omitempty"`
}

// PerfReport 是 seckillload 完成后生成的完整报告。
// 后端在 job 完成时持久化到 report.json，前端通过 GET /jobs/:id 直接获取。
type PerfReport struct {
	GeneratedAt string          `json:"generated_at"`
	Config      json.RawMessage `json:"config"`
	Summary     json.RawMessage `json:"summary"`
	Samples     []PerfProgress  `json:"samples"`
}

// ─── seckillload stdout JSON 解析 ───

// seckillloadProgress 是 seckillload progress JSON 的解析目标。
type seckillloadProgress struct {
	Kind    string `json:"kind"`
	Summary struct {
		Total                      int                `json:"total"`
		Success                    int                `json:"success"`
		RPS                        float64            `json:"rps"`
		SuccessRate                float64            `json:"success_rate"`
		NetworkErrorRate           float64            `json:"network_error_rate"`
		LatencyP95Ms               float64            `json:"latency_p95_ms"`
		BusinessCode               map[string]float64 `json:"business_code"`
		PromQps                    *float64           `json:"promQps"`
		PromP99LatencyMs           *float64           `json:"promP99LatencyMs"`
		PromErrorRate              *float64           `json:"promErrorRate"`
		PurchaseKafkaPublishRate   *float64           `json:"purchaseKafkaPublishRate"`
		PurchaseKafkaPublishFailed *float64           `json:"purchaseKafkaPublishFailed"`
		OrderStateConsumeRate      *float64           `json:"orderStateConsumeRate"`
	} `json:"summary"`
	GeneratedAt string          `json:"generated_at"`
	Config      json.RawMessage `json:"config"`
}

// knownRejectCodes 已知的正常业务拒绝 code。
var knownRejectCodes = map[string]bool{
	"SECKILL_OUT_OF_STOCK":           true,
	"SECKILL_PURCHASE_CONFLICT":      true,
	"SECKILL_LIMIT_EXCEEDED":         true,
	"SECKILL_ACTIVITY_ENDED":         true,
	"SECKILL_ACTIVITY_NOT_STARTED":   true,
	"SECKILL_ACTIVITY_NOT_PUBLISHED": true,
	"SECKILL_ACTIVITY_NOT_FOUND":     true,
	"SECKILL_ITEM_NOT_FOUND":         true,
}

// ParseSeckillloadLine 尝试解析 seckillload 的 stdout JSON 行。
// 返回:
//   - progress: 非 nil 表示解析为 progress 点
//   - report:   非 nil 表示解析为 final report
//   - 两者都为 nil 表示该行不是 seckillload JSON
func ParseSeckillloadLine(raw string) (progress *PerfProgress, report *PerfReport) {
	if len(raw) == 0 || raw[0] != '{' {
		return nil, nil
	}

	var probe seckillloadProgress
	if err := json.Unmarshal([]byte(raw), &probe); err != nil {
		return nil, nil
	}

	// 没有 summary 字段 → 不是有效 payload
	if probe.Summary.Total == 0 && probe.Summary.RPS == 0 {
		return nil, nil
	}

	ts := parseTimestamp(probe.GeneratedAt)
	label := formatTimeLabel(ts)

	// 分类 business_code
	rejectRate, systemErrorRate := classifyBusinessCodes(probe.Summary.BusinessCode, probe.Summary.Total)

	// 库存扣减率
	stockDeductRate := calcStockDeductionRate(probe.Summary.Success, probe.Summary.SuccessRate, probe.Summary.BusinessCode)

	pp := &PerfProgress{
		Timestamp:                  ts,
		Label:                      label,
		QPS:                        probe.Summary.RPS,
		P95LatencyMs:               probe.Summary.LatencyP95Ms,
		SuccessRate:                asPercent(probe.Summary.SuccessRate),
		RejectRate:                 rejectRate,
		SystemErrorRate:            systemErrorRate,
		NetworkErrorRate:           asPercent(probe.Summary.NetworkErrorRate),
		StockDeductRate:            stockDeductRate,
		PromQps:                    cloneFloatPtr(probe.Summary.PromQps),
		PromP99LatencyMs:           cloneFloatPtr(probe.Summary.PromP99LatencyMs),
		PromErrorRate:              cloneFloatPtr(probe.Summary.PromErrorRate),
		PurchaseKafkaPublishRate:   cloneFloatPtr(probe.Summary.PurchaseKafkaPublishRate),
		PurchaseKafkaPublishFailed: cloneFloatPtr(probe.Summary.PurchaseKafkaPublishFailed),
		OrderStateConsumeRate:      cloneFloatPtr(probe.Summary.OrderStateConsumeRate),
	}

	if probe.Kind == "progress" {
		return pp, nil
	}

	// final report（没有 kind 或 kind 不是 progress）
	rpt := &PerfReport{
		GeneratedAt: probe.GeneratedAt,
		Config:      probe.Config,
	}
	// 保留原始 summary
	if summaryRaw, err := json.Marshal(probe.Summary); err == nil {
		rpt.Summary = summaryRaw
	}
	return pp, rpt
}

// ─── 辅助函数 ───

func parseTimestamp(isoStr string) int64 {
	if isoStr == "" {
		return timeNowUnixMilli()
	}
	for _, layout := range []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05.999999999Z",
	} {
		if t, err := timeParse(layout, isoStr); err == nil {
			return t.UnixMilli()
		}
	}
	return timeNowUnixMilli()
}

func formatTimeLabel(tsMs int64) string {
	return timeUnixMilli(tsMs).Format("15:04:05")
}

func classifyBusinessCodes(codes map[string]float64, total int) (rejectRate, systemErrorRate float64) {
	if total <= 0 || len(codes) == 0 {
		return 0, 0
	}
	var rejectCount, systemErrorCount float64
	for code, count := range codes {
		if code == "OK" || code == "<empty>" {
			continue
		}
		if knownRejectCodes[code] {
			rejectCount += count
		} else {
			systemErrorCount += count
		}
	}
	rejectRate = roundPercent(rejectCount / float64(total))
	systemErrorRate = roundPercent(systemErrorCount / float64(total))
	return
}

func calcStockDeductionRate(success int, successRate float64, codes map[string]float64) float64 {
	outOfStock := codes["SECKILL_OUT_OF_STOCK"]
	purchaseConflict := codes["SECKILL_PURCHASE_CONFLICT"]
	denominator := float64(success) + outOfStock + purchaseConflict
	if denominator <= 0 {
		return asPercent(successRate)
	}
	return roundPercent(float64(success) / denominator)
}

func asPercent(v float64) float64 {
	if v <= 1 {
		return roundTo2(v * 100)
	}
	return roundTo2(v)
}

func roundPercent(v float64) float64 {
	return roundTo2(v * 100)
}

func roundTo2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}

func cloneFloatPtr(v *float64) *float64 {
	if v == nil {
		return nil
	}
	cloned := *v
	return &cloned
}

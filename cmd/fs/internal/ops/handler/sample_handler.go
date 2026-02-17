package handler

import (
	"net/http"
	"strconv"
	"time"

	"flashsale/cmd/fs/internal/ops/service"
)

// SampleHandler 处理采样数据的 HTTP 请求。
type SampleHandler struct {
	Store *service.SampleStore
}

// GetSamples 查询采样数据。
// GET /api/v1/samples?range=1h&step=2s
// GET /api/v1/samples?start=<ms>&end=<ms>&step=2s  (绝对时间范围)
//
// range: 1h | 3h | 6h | 12h | 24h | 3d | 7d
// step:  2s | 5s | 15s | 1m | 5m  (可选，默认根据 range 自动选择)
func (h *SampleHandler) GetSamples(w http.ResponseWriter, r *http.Request) {
	var startMs, endMs int64
	var rangeDuration time.Duration

	// 优先使用绝对时间范围
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	if startStr != "" && endStr != "" {
		var err error
		startMs, err = strconv.ParseInt(startStr, 10, 64)
		if err != nil {
			WriteErr(w, http.StatusBadRequest, "invalid start: "+startStr)
			return
		}
		endMs, err = strconv.ParseInt(endStr, 10, 64)
		if err != nil {
			WriteErr(w, http.StatusBadRequest, "invalid end: "+endStr)
			return
		}
		if endMs <= startMs {
			WriteErr(w, http.StatusBadRequest, "end must be after start")
			return
		}
		rangeDuration = time.Duration(endMs-startMs) * time.Millisecond
	} else {
		rangeStr := r.URL.Query().Get("range")
		if rangeStr == "" {
			rangeStr = "1h"
		}
		var err error
		rangeDuration, err = parseRange(rangeStr)
		if err != nil {
			WriteErr(w, http.StatusBadRequest, "invalid range: "+rangeStr)
			return
		}
		now := time.Now()
		endMs = now.UnixMilli()
		startMs = now.Add(-rangeDuration).UnixMilli()
	}

	// 解析 step
	stepStr := r.URL.Query().Get("step")
	var stepMs int64
	var err error
	if stepStr != "" {
		stepMs, err = parseStepMs(stepStr)
		if err != nil {
			WriteErr(w, http.StatusBadRequest, "invalid step: "+stepStr)
			return
		}
	} else {
		stepMs = autoStep(rangeDuration)
	}

	samples, err := h.Store.Query(startMs, endMs, stepMs)
	if err != nil {
		WriteErr(w, http.StatusInternalServerError, "query samples failed: "+err.Error())
		return
	}

	if samples == nil {
		samples = []service.MonitorSample{}
	}

	WriteOK(w, map[string]any{
		"samples": samples,
		"meta": map[string]any{
			"step":    stepStr,
			"stepMs":  stepMs,
			"count":   len(samples),
			"startMs": startMs,
			"endMs":   endMs,
		},
	})
}

// GetAvailableDays 返回有数据的日期列表。
// GET /api/v1/samples/days
func (h *SampleHandler) GetAvailableDays(w http.ResponseWriter, _ *http.Request) {
	days := h.Store.ListAvailableDays()
	if days == nil {
		days = []string{}
	}
	WriteOK(w, map[string]any{"days": days})
}

// ─── 辅助函数 ───

var rangeMap = map[string]time.Duration{
	"1h":  time.Hour,
	"3h":  3 * time.Hour,
	"6h":  6 * time.Hour,
	"12h": 12 * time.Hour,
	"24h": 24 * time.Hour,
	"3d":  3 * 24 * time.Hour,
	"7d":  7 * 24 * time.Hour,
}

func parseRange(s string) (time.Duration, error) {
	if d, ok := rangeMap[s]; ok {
		return d, nil
	}
	return 0, http.ErrNotSupported
}

func parseStepMs(s string) (int64, error) {
	// 支持格式: 2s, 5s, 15s, 1m, 5m
	if len(s) < 2 {
		return 0, http.ErrNotSupported
	}
	unit := s[len(s)-1]
	numStr := s[:len(s)-1]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, err
	}
	switch unit {
	case 's':
		return int64(num) * 1000, nil
	case 'm':
		return int64(num) * 60 * 1000, nil
	default:
		return 0, http.ErrNotSupported
	}
}

// autoStep 根据时间范围自动选择降采样步长。
func autoStep(d time.Duration) int64 {
	switch {
	case d <= time.Hour:
		return 0 // 原始粒度 (2s)
	case d <= 3*time.Hour:
		return 5000 // 5s
	case d <= 6*time.Hour:
		return 15000 // 15s
	case d <= 24*time.Hour:
		return 60000 // 1m
	default:
		return 300000 // 5m
	}
}

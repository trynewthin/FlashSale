package api

import (
	"net/http"
	"strconv"
	"time"

	"flashsale/ops/backend/store"
)

type SampleHandler struct {
	Store     *store.SampleStore
	Collector *store.SampleCollector
}

func (h *SampleHandler) SyncNow(w http.ResponseWriter, r *http.Request) {
	if h.Collector != nil {
		h.Collector.TriggerNow()
	}
	WriteOK(w, map[string]string{"message": "sync triggered"})
}

func (h *SampleHandler) GetSamples(w http.ResponseWriter, r *http.Request) {
	var startMs, endMs int64
	var rangeDuration time.Duration
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
		samples = []store.MonitorSample{}
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

func (h *SampleHandler) GetAvailableDays(w http.ResponseWriter, _ *http.Request) {
	days := h.Store.ListAvailableDays()
	if days == nil {
		days = []string{}
	}
	WriteOK(w, map[string]any{"days": days})
}

var rangeMap = map[string]time.Duration{
	"1m":  time.Minute,
	"5m":  5 * time.Minute,
	"10m": 10 * time.Minute,
	"30m": 30 * time.Minute,
	"1h":  time.Hour,
	"5h":  5 * time.Hour,
	"12h": 12 * time.Hour,
	"1d":  24 * time.Hour,
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
	if len(s) < 2 {
		return 0, http.ErrNotSupported
	}
	unit := s[len(s)-1]
	num, err := strconv.Atoi(s[:len(s)-1])
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

func autoStep(d time.Duration) int64 {
	switch {
	case d <= 10*time.Minute:
		return 0 // 原始精度，不降采样
	case d <= 30*time.Minute:
		return 2000
	case d <= time.Hour:
		return 5000
	case d <= 5*time.Hour:
		return 15000
	case d <= 24*time.Hour:
		return 60000
	default:
		return 300000
	}
}

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"
)

// ─── 统计结果 ───

type summary struct {
	Total         int
	Success       int
	UniqueOrders  int
	NetworkErrors int // 真正的网络错误（不含 local_drop）
	LocalDrops    int // 开环限流丢弃数（不算网络错误）
	TrackAccepted int
	TrackDegraded int
	Elapsed       time.Duration
	RPS           float64
	MeanLatency   time.Duration
	P50Latency    time.Duration
	P95Latency    time.Duration
	P99Latency    time.Duration
	MaxLatency    time.Duration
	HTTPStats     map[int]int
	CodeStats     map[string]int
	ErrorStats    map[string]int
	ErrorSamples  map[string]string
}

// ─── 汇总 ───

func summarize(results []requestResult, elapsed time.Duration) summary {
	s := summary{
		Total:        len(results),
		Elapsed:      elapsed,
		HTTPStats:    make(map[int]int),
		CodeStats:    make(map[string]int),
		ErrorStats:   make(map[string]int),
		ErrorSamples: make(map[string]string),
	}
	if len(results) == 0 {
		return s
	}

	latencies := make([]time.Duration, 0, len(results))
	uniqueOrders := make(map[string]struct{})
	var sumLatency time.Duration
	for _, r := range results {
		latencies = append(latencies, r.Latency)
		sumLatency += r.Latency
		if r.Latency > s.MaxLatency {
			s.MaxLatency = r.Latency
		}
		s.HTTPStats[r.HTTPStatus]++
		if strings.TrimSpace(r.Code) != "" {
			s.CodeStats[r.Code]++
		} else {
			s.CodeStats["<empty>"]++
		}
		if r.Err != nil {
			errKind := classifyNetworkError(r.Err)
			s.ErrorStats[errKind]++
			if _, ok := s.ErrorSamples[errKind]; !ok {
				s.ErrorSamples[errKind] = strings.TrimSpace(r.Err.Error())
			}
			// local_drop 不计入 NetworkErrors
			if errors.Is(r.Err, errOpenModelDropped) {
				s.LocalDrops++
			} else {
				s.NetworkErrors++
			}
		}
		if r.TrackAcceptedKnow {
			if r.TrackAccepted {
				s.TrackAccepted++
			} else {
				s.TrackDegraded++
			}
		}
		if r.Err == nil && r.HTTPStatus == http.StatusOK && r.Code == "OK" {
			if !r.TrackAcceptedKnow || r.TrackAccepted {
				s.Success++
			}
			if strings.TrimSpace(r.OrderNo) != "" {
				uniqueOrders[strings.TrimSpace(r.OrderNo)] = struct{}{}
			}
		}
	}
	s.UniqueOrders = len(uniqueOrders)
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	s.MeanLatency = sumLatency / time.Duration(len(latencies))
	s.P50Latency = percentile(latencies, 0.50)
	s.P95Latency = percentile(latencies, 0.95)
	s.P99Latency = percentile(latencies, 0.99)
	if elapsed > 0 {
		s.RPS = float64(len(results)) / elapsed.Seconds()
	}
	return s
}

func percentile(data []time.Duration, p float64) time.Duration {
	if len(data) == 0 {
		return 0
	}
	if p <= 0 {
		return data[0]
	}
	if p >= 1 {
		return data[len(data)-1]
	}
	idx := int(float64(len(data)-1) * p)
	return data[idx]
}

// ─── 错误分类 ───

func classifyNetworkError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, errOpenModelDropped) {
		return "local_drop"
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(msg, "deadline exceeded") || strings.Contains(msg, "timeout"):
		return "timeout"
	case strings.Contains(msg, "cannot assign requested address"):
		return "local_ephemeral_exhaustion"
	case strings.Contains(msg, "forcibly closed") || strings.Contains(msg, "connection reset"):
		return "conn_reset"
	case strings.Contains(msg, "dial tcp"):
		if opErr := (*net.OpError)(nil); errors.As(err, &opErr) {
			if opErr.Op == "dial" {
				return "dial_error"
			}
		}
		return "dial_error"
	case strings.Contains(msg, "connection refused"):
		return "conn_refused"
	case strings.Contains(msg, "no such host"):
		return "dns"
	case strings.Contains(msg, "tls") && strings.Contains(msg, "handshake"):
		return "tls_handshake"
	default:
		if len(msg) > 96 {
			return msg[:96]
		}
		if msg == "" {
			return "unknown"
		}
		return msg
	}
}

// ─── 输出 ───

// reportPayload 是压测机读输出，用于脚本门禁判定和报告沉淀。
type reportPayload struct {
	Kind        string         `json:"kind,omitempty"`
	GeneratedAt string         `json:"generated_at"`
	Config      reportConfig   `json:"config"`
	Summary     summaryPayload `json:"summary"`
}

type reportConfig struct {
	Scenario       string `json:"scenario"`
	BaseURL        string `json:"base_url"`
	ActivityID     int64  `json:"activity_id"`
	ActivityItemID int64  `json:"item_id"`
	Requests       int    `json:"requests"`
	Concurrency    int    `json:"concurrency"`
	OpenRate       int    `json:"open_rate,omitempty"`
	OpenDurationMs int64  `json:"open_duration_ms,omitempty"`
	TimeoutMs      int64  `json:"timeout_ms"`
}

type summaryPayload struct {
	Total            int               `json:"total"`
	Success          int               `json:"success"`
	SuccessRate      float64           `json:"success_rate"`
	UniqueOrders     int               `json:"unique_orders"`
	NetworkErrors    int               `json:"network_errors"`
	NetworkErrorRate float64           `json:"network_error_rate"`
	LocalDrops       int               `json:"local_drops"`
	LocalDropRate    float64           `json:"local_drop_rate"`
	TrackAccepted    int               `json:"track_accepted"`
	TrackDegraded    int               `json:"track_degraded"`
	TrackAcceptRate  float64           `json:"track_accept_rate"`
	ElapsedMs        int64             `json:"elapsed_ms"`
	RPS              float64           `json:"rps"`
	MeanLatencyMs    float64           `json:"latency_mean_ms"`
	P50LatencyMs     float64           `json:"latency_p50_ms"`
	P95LatencyMs     float64           `json:"latency_p95_ms"`
	P99LatencyMs     float64           `json:"latency_p99_ms"`
	MaxLatencyMs     float64           `json:"latency_max_ms"`
	HTTPStats        map[int]int       `json:"http_status"`
	CodeStats        map[string]int    `json:"business_code"`
	ErrorStats       map[string]int    `json:"network_error_kind"`
	ErrorSamples     map[string]string `json:"network_error_sample,omitempty"`
}

func printSummaryJSON(cfg runConfig, s summary) {
	printSummaryJSONWithKind(cfg, s, "")
}

func printProgressJSON(cfg runConfig, s summary) {
	printSummaryJSONWithKind(cfg, s, "progress")
}

func printSummaryJSONWithKind(cfg runConfig, s summary, kind string) {
	payload := reportPayload{
		Kind:        strings.TrimSpace(kind),
		GeneratedAt: time.Now().Format(time.RFC3339),
		Config: reportConfig{
			Scenario:       cfg.Scenario,
			BaseURL:        cfg.BaseURL,
			ActivityID:     cfg.ActivityID,
			ActivityItemID: cfg.ActivityItemID,
			Requests:       cfg.Requests,
			Concurrency:    cfg.Concurrency,
			OpenRate:       cfg.OpenRate,
			OpenDurationMs: cfg.OpenDuration.Milliseconds(),
			TimeoutMs:      cfg.Timeout.Milliseconds(),
		},
		Summary: summaryPayload{
			Total:            s.Total,
			Success:          s.Success,
			SuccessRate:      ratio(s.Success, s.Total),
			UniqueOrders:     s.UniqueOrders,
			NetworkErrors:    s.NetworkErrors,
			NetworkErrorRate: ratio(s.NetworkErrors, s.Total),
			LocalDrops:       s.LocalDrops,
			LocalDropRate:    ratio(s.LocalDrops, s.Total),
			TrackAccepted:    s.TrackAccepted,
			TrackDegraded:    s.TrackDegraded,
			TrackAcceptRate:  ratio(s.TrackAccepted, s.TrackAccepted+s.TrackDegraded),
			ElapsedMs:        s.Elapsed.Milliseconds(),
			RPS:              s.RPS,
			MeanLatencyMs:    float64(s.MeanLatency) / float64(time.Millisecond),
			P50LatencyMs:     float64(s.P50Latency) / float64(time.Millisecond),
			P95LatencyMs:     float64(s.P95Latency) / float64(time.Millisecond),
			P99LatencyMs:     float64(s.P99Latency) / float64(time.Millisecond),
			MaxLatencyMs:     float64(s.MaxLatency) / float64(time.Millisecond),
			HTTPStats:        s.HTTPStats,
			CodeStats:        s.CodeStats,
			ErrorStats:       s.ErrorStats,
			ErrorSamples:     s.ErrorSamples,
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		fatalf("marshal json summary failed: %v", err)
	}
	fmt.Println(string(raw))
}

func printSummary(cfg runConfig, s summary) {
	fmt.Println("=== Seckill Pressure Summary ===")
	fmt.Printf("Scenario          : %s\n", cfg.Scenario)
	fmt.Printf("Base URL          : %s\n", cfg.BaseURL)
	fmt.Printf("Activity/Item     : %d / %d\n", cfg.ActivityID, cfg.ActivityItemID)
	if cfg.isOpenModel() {
		fmt.Printf("Open Rate/Duration: %d req/s / %s\n", cfg.OpenRate, cfg.OpenDuration)
	}
	fmt.Printf("Requests/Conc     : %d / %d\n", cfg.Requests, cfg.Concurrency)
	fmt.Printf("Elapsed           : %s\n", s.Elapsed)
	fmt.Printf("RPS               : %.2f\n", s.RPS)
	fmt.Printf("Total             : %d\n", s.Total)
	fmt.Printf("Success           : %d\n", s.Success)
	if cfg.Scenario == scenarioTrackStress {
		fmt.Printf("Track Accepted    : %d\n", s.TrackAccepted)
		fmt.Printf("Track Degraded    : %d\n", s.TrackDegraded)
	}
	if cfg.Scenario == scenarioPurchaseStress || cfg.Scenario == scenarioIdempotency {
		fmt.Printf("Unique Orders     : %d\n", s.UniqueOrders)
	}
	fmt.Printf("Network Errors    : %d\n", s.NetworkErrors)
	if s.LocalDrops > 0 {
		fmt.Printf("Local Drops       : %d (not counted as errors)\n", s.LocalDrops)
	}
	fmt.Printf("Latency mean/p50  : %s / %s\n", s.MeanLatency, s.P50Latency)
	fmt.Printf("Latency p95/p99   : %s / %s\n", s.P95Latency, s.P99Latency)
	fmt.Printf("Latency max       : %s\n", s.MaxLatency)

	fmt.Println("HTTP Status Stats:")
	printIntMap(s.HTTPStats)
	fmt.Println("Business Code Stats:")
	printStringMap(s.CodeStats)
	if len(s.ErrorStats) > 0 {
		fmt.Println("Network Error Stats:")
		printStringMap(s.ErrorStats)
		fmt.Println("Network Error Samples:")
		printStringSamples(s.ErrorSamples)
	}
}

func printIntMap(m map[int]int) {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		fmt.Printf("  %d: %d\n", k, m[k])
	}
}

func printStringMap(m map[string]int) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %s: %d\n", k, m[k])
	}
}

func printStringSamples(m map[string]string) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %s: %s\n", k, m[k])
	}
}

func ratio(numerator, denominator int) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

// ─── 断言 ───

// assertScenario 检查测试结果是否满足断言条件，返回 nil 表示通过。
// NetworkErrors 仅统计真正的网络错误（timeout/conn_reset 等），
// local_drop（开环限流丢弃）不计入。
func assertScenario(cfg runConfig, s summary) error {
	if cfg.MaxNetworkErrors >= 0 && s.NetworkErrors > cfg.MaxNetworkErrors {
		return fmt.Errorf("network errors exceeded: got=%d max=%d (local_drops=%d not counted)", s.NetworkErrors, cfg.MaxNetworkErrors, s.LocalDrops)
	}
	if cfg.ExpectMaxSuccess >= 0 {
		value := s.Success
		label := "success count"
		if cfg.Scenario == scenarioIdempotency {
			value = s.UniqueOrders
			label = "unique orders"
		}
		if value > cfg.ExpectMaxSuccess {
			return fmt.Errorf("%s exceeded: got=%d expect-max=%d", label, value, cfg.ExpectMaxSuccess)
		}
	}
	if cfg.Scenario == scenarioIdempotency && s.UniqueOrders > 1 {
		return fmt.Errorf("idempotency scenario violated: unique orders=%d should <= 1", s.UniqueOrders)
	}
	return nil
}

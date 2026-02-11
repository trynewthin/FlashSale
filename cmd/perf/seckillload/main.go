// seckillload 提供秒杀链路压测与典型问题场景测试入口。
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	scenarioPurchaseStress = "purchase-stress"
	scenarioIdempotency    = "idempotency"
	scenarioTrackStress    = "track-stress"
	outputText             = "text"
	outputJSON             = "json"
)

type envelope struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type requestResult struct {
	Latency           time.Duration
	HTTPStatus        int
	Code              string
	OrderNo           string
	TrackAccepted     bool
	TrackAcceptedKnow bool
	Err               error
}

type runConfig struct {
	Scenario         string
	BaseURL          string
	ActivityID       int64
	ActivityItemID   int64
	Concurrency      int
	Requests         int
	Quantity         int64
	EventType        string
	ClientIDPrefix   string
	IdempotencyGroup string
	ExpectMaxSuccess int
	MaxNetworkErrors int
	Timeout          time.Duration
	MaxIdleConns     int
	MaxIdlePerHost   int
	MaxConnsPerHost  int
	DisableKeepAlive bool
	Token            string
	TokenFile        string
	Output           string
}

// reportPayload 是压测机读输出，用于脚本门禁判定和报告沉淀。
type reportPayload struct {
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
	TimeoutMs      int64  `json:"timeout_ms"`
}

type summaryPayload struct {
	Total            int               `json:"total"`
	Success          int               `json:"success"`
	SuccessRate      float64           `json:"success_rate"`
	UniqueOrders     int               `json:"unique_orders"`
	NetworkErrors    int               `json:"network_errors"`
	NetworkErrorRate float64           `json:"network_error_rate"`
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

func main() {
	cfg := parseFlags()
	if err := validateConfig(cfg); err != nil {
		fatalf("invalid config: %v", err)
	}

	tokens, err := loadTokens(cfg)
	if err != nil {
		fatalf("load tokens failed: %v", err)
	}

	client := newHTTPClient(cfg)
	start := time.Now()
	results := runScenario(client, cfg, tokens)
	elapsed := time.Since(start)

	summary := summarize(results, elapsed)
	if cfg.Output == outputJSON {
		printSummaryJSON(cfg, summary)
	} else {
		printSummary(cfg, summary)
	}

	if err := assertScenario(cfg, summary); err != nil {
		fatalf("%v", err)
	}
}

func parseFlags() runConfig {
	cfg := runConfig{}
	flag.StringVar(&cfg.Scenario, "scenario", scenarioPurchaseStress, "压测场景: purchase-stress|idempotency|track-stress")
	flag.StringVar(&cfg.BaseURL, "base-url", envOrDefault("FLASHSALE_BASE_URL", "http://127.0.0.1:8082"), "用户网关地址")
	flag.Int64Var(&cfg.ActivityID, "activity-id", envInt64("FLASHSALE_ACTIVITY_ID", 0), "秒杀活动ID")
	flag.Int64Var(&cfg.ActivityItemID, "item-id", envInt64("FLASHSALE_ACTIVITY_ITEM_ID", 0), "活动商品ID")
	flag.IntVar(&cfg.Concurrency, "concurrency", envInt("FLASHSALE_PRESSURE_CONCURRENCY", 100), "并发数")
	flag.IntVar(&cfg.Requests, "requests", envInt("FLASHSALE_PRESSURE_REQUESTS", 1000), "请求总数")
	flag.Int64Var(&cfg.Quantity, "quantity", envInt64("FLASHSALE_PRESSURE_QUANTITY", 1), "购买件数")
	flag.StringVar(&cfg.EventType, "event-type", envOrDefault("FLASHSALE_TRACK_EVENT_TYPE", "pv"), "埋点事件类型")
	flag.StringVar(&cfg.ClientIDPrefix, "client-id-prefix", envOrDefault("FLASHSALE_CLIENT_ID_PREFIX", "perf-client"), "匿名埋点 client_id 前缀")
	flag.StringVar(&cfg.IdempotencyGroup, "idempotency-group", envOrDefault("FLASHSALE_IDEMPOTENCY_GROUP", "same-key-group"), "幂等场景固定分组键")
	flag.IntVar(&cfg.ExpectMaxSuccess, "expect-max-success", envInt("FLASHSALE_EXPECT_MAX_SUCCESS", -1), "断言上限（普通场景按 Success，幂等场景按 Unique Orders；<0 表示不校验）")
	flag.IntVar(&cfg.MaxNetworkErrors, "max-network-errors", envInt("FLASHSALE_MAX_NETWORK_ERRORS", 0), "允许的网络错误上限")
	flag.DurationVar(&cfg.Timeout, "timeout", envDuration("FLASHSALE_HTTP_TIMEOUT", 3*time.Second), "单请求超时")
	flag.IntVar(&cfg.MaxIdleConns, "max-idle-conns", envInt("FLASHSALE_HTTP_MAX_IDLE_CONNS", 1024), "HTTP transport MaxIdleConns")
	flag.IntVar(&cfg.MaxIdlePerHost, "max-idle-conns-per-host", envInt("FLASHSALE_HTTP_MAX_IDLE_CONNS_PER_HOST", 512), "HTTP transport MaxIdleConnsPerHost")
	flag.IntVar(&cfg.MaxConnsPerHost, "max-conns-per-host", envInt("FLASHSALE_HTTP_MAX_CONNS_PER_HOST", 0), "HTTP transport MaxConnsPerHost，0 表示不限制")
	flag.BoolVar(&cfg.DisableKeepAlive, "disable-keepalive", envBool("FLASHSALE_HTTP_DISABLE_KEEPALIVE", false), "禁用 HTTP keepalive")
	flag.StringVar(&cfg.Token, "token", strings.TrimSpace(os.Getenv("FLASHSALE_USER_TOKEN")), "用户 JWT")
	flag.StringVar(&cfg.TokenFile, "token-file", strings.TrimSpace(os.Getenv("FLASHSALE_TOKENS_FILE")), "token 文件路径（每行一个）")
	flag.StringVar(&cfg.Output, "output", envOrDefault("FLASHSALE_PRESSURE_OUTPUT", outputText), "输出格式: text|json")
	flag.Parse()
	return cfg
}

func newHTTPClient(cfg runConfig) *http.Client {
	maxIdle := cfg.MaxIdleConns
	if maxIdle < 0 {
		maxIdle = 0
	}
	maxIdlePerHost := cfg.MaxIdlePerHost
	if maxIdlePerHost < 0 {
		maxIdlePerHost = 0
	}
	maxConnsPerHost := cfg.MaxConnsPerHost
	if maxConnsPerHost < 0 {
		maxConnsPerHost = 0
	}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          maxIdle,
		MaxIdleConnsPerHost:   maxIdlePerHost,
		MaxConnsPerHost:       maxConnsPerHost,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     cfg.DisableKeepAlive,
	}
	return &http.Client{Timeout: cfg.Timeout, Transport: transport}
}

func validateConfig(cfg runConfig) error {
	switch cfg.Scenario {
	case scenarioPurchaseStress, scenarioIdempotency, scenarioTrackStress:
	default:
		return fmt.Errorf("unsupported scenario: %s", cfg.Scenario)
	}
	if cfg.ActivityID <= 0 || cfg.ActivityItemID <= 0 {
		return errors.New("activity-id 和 item-id 必须大于 0")
	}
	if cfg.Concurrency <= 0 {
		return errors.New("concurrency 必须大于 0")
	}
	if cfg.Requests <= 0 {
		return errors.New("requests 必须大于 0")
	}
	if cfg.Quantity <= 0 {
		return errors.New("quantity 必须大于 0")
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return errors.New("base-url 不能为空")
	}
	if cfg.Timeout <= 0 {
		return errors.New("timeout 必须大于 0")
	}
	switch cfg.Output {
	case outputText, outputJSON:
	default:
		return errors.New("output 仅支持 text 或 json")
	}
	return nil
}

func loadTokens(cfg runConfig) ([]string, error) {
	list := make([]string, 0, 16)
	if token := strings.TrimSpace(cfg.Token); token != "" {
		list = append(list, token)
	}
	if file := strings.TrimSpace(cfg.TokenFile); file != "" {
		f, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			v := strings.TrimSpace(scanner.Text())
			if v == "" || strings.HasPrefix(v, "#") {
				continue
			}
			list = append(list, v)
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}
	if (cfg.Scenario == scenarioPurchaseStress || cfg.Scenario == scenarioIdempotency) && len(list) == 0 {
		return nil, errors.New("购买场景必须提供 token 或 token-file")
	}
	return list, nil
}

func runScenario(client *http.Client, cfg runConfig, tokens []string) []requestResult {
	results := make([]requestResult, 0, cfg.Requests)
	jobs := make(chan int, cfg.Requests)
	out := make(chan requestResult, cfg.Requests)

	var wg sync.WaitGroup
	workerCount := cfg.Concurrency
	if workerCount > cfg.Requests {
		workerCount = cfg.Requests
	}
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for idx := range jobs {
				out <- executeOne(client, cfg, tokens, workerID, idx)
			}
		}(i)
	}

	for i := 0; i < cfg.Requests; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	close(out)

	for r := range out {
		results = append(results, r)
	}
	return results
}

func executeOne(client *http.Client, cfg runConfig, tokens []string, workerID, idx int) requestResult {
	start := time.Now()
	var (
		status        int
		code          string
		order         string
		trackAccepted bool
		trackKnown    bool
		err           error
	)

	switch cfg.Scenario {
	case scenarioPurchaseStress:
		idem := fmt.Sprintf("perf-purchase-%d-%d", time.Now().UnixNano(), idx)
		token := tokens[idx%len(tokens)]
		status, code, order, err = doPurchase(client, cfg, token, idem)
	case scenarioIdempotency:
		idem := fmt.Sprintf("perf-idem-%s", strings.TrimSpace(cfg.IdempotencyGroup))
		// 幂等场景必须固定同一用户，否则会被“多用户多订单”误判为幂等失效。
		token := tokens[0]
		status, code, order, err = doPurchase(client, cfg, token, idem)
	case scenarioTrackStress:
		idem := fmt.Sprintf("perf-track-%d-%d", time.Now().UnixNano(), idx)
		clientID := fmt.Sprintf("%s-%d-%d", strings.TrimSpace(cfg.ClientIDPrefix), workerID, idx)
		status, code, trackAccepted, err = doTrack(client, cfg, idem, clientID)
		trackKnown = true
	}
	return requestResult{
		Latency:           time.Since(start),
		HTTPStatus:        status,
		Code:              code,
		OrderNo:           order,
		TrackAccepted:     trackAccepted,
		TrackAcceptedKnow: trackKnown,
		Err:               err,
	}
}

func doPurchase(client *http.Client, cfg runConfig, token, idempotencyKey string) (int, string, string, error) {
	path := fmt.Sprintf("%s/api/v1/seckill/activities/%d/purchase", strings.TrimRight(cfg.BaseURL, "/"), cfg.ActivityID)
	body := map[string]any{
		"activity_item_id": cfg.ActivityItemID,
		"quantity":         cfg.Quantity,
		"idempotency_key":  idempotencyKey,
	}
	statusCode, code, data, err := doJSON(client, http.MethodPost, path, token, body)
	if err != nil {
		return statusCode, code, "", err
	}
	orderNo := ""
	if len(data) > 0 {
		var order struct {
			OrderNo string `json:"order_no"`
		}
		if err := json.Unmarshal(data, &order); err == nil {
			orderNo = strings.TrimSpace(order.OrderNo)
		}
	}
	return statusCode, code, orderNo, nil
}

func doTrack(client *http.Client, cfg runConfig, idempotencyKey, clientID string) (int, string, bool, error) {
	path := fmt.Sprintf("%s/api/v1/seckill/activities/%d/track", strings.TrimRight(cfg.BaseURL, "/"), cfg.ActivityID)
	body := map[string]any{
		"activity_item_id": cfg.ActivityItemID,
		"event_type":       cfg.EventType,
		"client_id":        clientID,
		"idempotency_key":  idempotencyKey,
		"occurred_at_unix": time.Now().Unix(),
	}
	statusCode, code, data, err := doJSON(client, http.MethodPost, path, "", body)
	if err != nil {
		return statusCode, code, false, err
	}
	accepted := false
	if len(data) > 0 {
		var track struct {
			Accepted bool `json:"accepted"`
		}
		if err := json.Unmarshal(data, &track); err == nil {
			accepted = track.Accepted
		}
	}
	return statusCode, code, accepted, nil
}

func doJSON(client *http.Client, method, url, token string, payload any) (int, string, json.RawMessage, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, "", nil, err
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(raw))
	if err != nil {
		return 0, "", nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, "", nil, err
	}
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return resp.StatusCode, "", nil, err
	}
	return resp.StatusCode, strings.TrimSpace(env.Code), env.Data, nil
}

type summary struct {
	Total         int
	Success       int
	UniqueOrders  int
	NetworkErrors int
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
			s.NetworkErrors++
			errKind := classifyNetworkError(r.Err)
			s.ErrorStats[errKind]++
			if _, ok := s.ErrorSamples[errKind]; !ok {
				s.ErrorSamples[errKind] = strings.TrimSpace(r.Err.Error())
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

func printSummary(cfg runConfig, s summary) {
	fmt.Println("=== Seckill Pressure Summary ===")
	fmt.Printf("Scenario          : %s\n", cfg.Scenario)
	fmt.Printf("Base URL          : %s\n", cfg.BaseURL)
	fmt.Printf("Activity/Item     : %d / %d\n", cfg.ActivityID, cfg.ActivityItemID)
	fmt.Printf("Requests/Conc     : %d / %d\n", cfg.Requests, cfg.Concurrency)
	fmt.Printf("Elapsed           : %s\n", s.Elapsed)
	fmt.Printf("RPS               : %.2f\n", s.RPS)
	fmt.Printf("Success           : %d\n", s.Success)
	if cfg.Scenario == scenarioTrackStress {
		fmt.Printf("Track Accepted    : %d\n", s.TrackAccepted)
		fmt.Printf("Track Degraded    : %d\n", s.TrackDegraded)
	}
	if cfg.Scenario == scenarioPurchaseStress || cfg.Scenario == scenarioIdempotency {
		fmt.Printf("Unique Orders     : %d\n", s.UniqueOrders)
	}
	fmt.Printf("Network Errors    : %d\n", s.NetworkErrors)
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

func printSummaryJSON(cfg runConfig, s summary) {
	payload := reportPayload{
		GeneratedAt: time.Now().Format(time.RFC3339),
		Config: reportConfig{
			Scenario:       cfg.Scenario,
			BaseURL:        cfg.BaseURL,
			ActivityID:     cfg.ActivityID,
			ActivityItemID: cfg.ActivityItemID,
			Requests:       cfg.Requests,
			Concurrency:    cfg.Concurrency,
			TimeoutMs:      cfg.Timeout.Milliseconds(),
		},
		Summary: summaryPayload{
			Total:            s.Total,
			Success:          s.Success,
			SuccessRate:      ratio(s.Success, s.Total),
			UniqueOrders:     s.UniqueOrders,
			NetworkErrors:    s.NetworkErrors,
			NetworkErrorRate: ratio(s.NetworkErrors, s.Total),
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

func ratio(numerator, denominator int) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func classifyNetworkError(err error) string {
	if err == nil {
		return ""
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

func printIntMap(m map[int]int) {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		fmt.Printf("  %d => %d\n", k, m[k])
	}
}

func printStringMap(m map[string]int) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %s => %d\n", k, m[k])
	}
}

func printStringSamples(m map[string]string) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %s => %s\n", k, m[k])
	}
}

func assertScenario(cfg runConfig, s summary) error {
	if s.NetworkErrors > cfg.MaxNetworkErrors {
		return fmt.Errorf("network errors exceeded: got=%d max=%d", s.NetworkErrors, cfg.MaxNetworkErrors)
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

func envOrDefault(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func envInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	var out int
	if _, err := fmt.Sscanf(v, "%d", &out); err != nil {
		return fallback
	}
	return out
}

func envInt64(key string, fallback int64) int64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	var out int64
	if _, err := fmt.Sscanf(v, "%d", &out); err != nil {
		return fallback
	}
	return out
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func envBool(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
	os.Exit(1)
}

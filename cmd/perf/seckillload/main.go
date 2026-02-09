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
)

type envelope struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type requestResult struct {
	Latency    time.Duration
	HTTPStatus int
	Code       string
	OrderNo    string
	Err        error
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
	Token            string
	TokenFile        string
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

	client := &http.Client{Timeout: cfg.Timeout}
	start := time.Now()
	results := runScenario(client, cfg, tokens)
	elapsed := time.Since(start)

	summary := summarize(results, elapsed)
	printSummary(cfg, summary)

	if err := assertScenario(cfg, summary); err != nil {
		fatalf("%v", err)
	}
}

func parseFlags() runConfig {
	cfg := runConfig{}
	flag.StringVar(&cfg.Scenario, "scenario", scenarioPurchaseStress, "压测场景: purchase-stress|idempotency|track-stress")
	flag.StringVar(&cfg.BaseURL, "base-url", envOrDefault("FLASHSALE_BASE_URL", "http://127.0.0.1:8081"), "用户网关地址")
	flag.Int64Var(&cfg.ActivityID, "activity-id", envInt64("FLASHSALE_ACTIVITY_ID", 0), "秒杀活动ID")
	flag.Int64Var(&cfg.ActivityItemID, "item-id", envInt64("FLASHSALE_ACTIVITY_ITEM_ID", 0), "活动商品ID")
	flag.IntVar(&cfg.Concurrency, "concurrency", envInt("FLASHSALE_PRESSURE_CONCURRENCY", 100), "并发数")
	flag.IntVar(&cfg.Requests, "requests", envInt("FLASHSALE_PRESSURE_REQUESTS", 1000), "请求总数")
	flag.Int64Var(&cfg.Quantity, "quantity", envInt64("FLASHSALE_PRESSURE_QUANTITY", 1), "购买件数")
	flag.StringVar(&cfg.EventType, "event-type", envOrDefault("FLASHSALE_TRACK_EVENT_TYPE", "pv"), "埋点事件类型")
	flag.StringVar(&cfg.ClientIDPrefix, "client-id-prefix", envOrDefault("FLASHSALE_CLIENT_ID_PREFIX", "perf-client"), "匿名埋点 client_id 前缀")
	flag.StringVar(&cfg.IdempotencyGroup, "idempotency-group", envOrDefault("FLASHSALE_IDEMPOTENCY_GROUP", "same-key-group"), "幂等场景固定分组键")
	flag.IntVar(&cfg.ExpectMaxSuccess, "expect-max-success", envInt("FLASHSALE_EXPECT_MAX_SUCCESS", -1), "断言成功数上限（<0表示不校验）")
	flag.IntVar(&cfg.MaxNetworkErrors, "max-network-errors", envInt("FLASHSALE_MAX_NETWORK_ERRORS", 0), "允许的网络错误上限")
	flag.DurationVar(&cfg.Timeout, "timeout", envDuration("FLASHSALE_HTTP_TIMEOUT", 3*time.Second), "单请求超时")
	flag.StringVar(&cfg.Token, "token", strings.TrimSpace(os.Getenv("FLASHSALE_USER_TOKEN")), "用户 JWT")
	flag.StringVar(&cfg.TokenFile, "token-file", strings.TrimSpace(os.Getenv("FLASHSALE_TOKENS_FILE")), "token 文件路径（每行一个）")
	flag.Parse()
	return cfg
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
		status int
		code   string
		order  string
		err    error
	)

	switch cfg.Scenario {
	case scenarioPurchaseStress:
		idem := fmt.Sprintf("perf-purchase-%d-%d", time.Now().UnixNano(), idx)
		token := tokens[idx%len(tokens)]
		status, code, order, err = doPurchase(client, cfg, token, idem)
	case scenarioIdempotency:
		idem := fmt.Sprintf("perf-idem-%s", strings.TrimSpace(cfg.IdempotencyGroup))
		token := tokens[idx%len(tokens)]
		status, code, order, err = doPurchase(client, cfg, token, idem)
	case scenarioTrackStress:
		idem := fmt.Sprintf("perf-track-%d-%d", time.Now().UnixNano(), idx)
		clientID := fmt.Sprintf("%s-%d-%d", strings.TrimSpace(cfg.ClientIDPrefix), workerID, idx)
		status, code, _, err = doTrack(client, cfg, idem, clientID)
	}
	return requestResult{
		Latency:    time.Since(start),
		HTTPStatus: status,
		Code:       code,
		OrderNo:    order,
		Err:        err,
	}
}

func doPurchase(client *http.Client, cfg runConfig, token, idempotencyKey string) (int, string, string, error) {
	path := fmt.Sprintf("%s/api/v1/seckill/activities/%d/purchase", strings.TrimRight(cfg.BaseURL, "/"), cfg.ActivityID)
	body := map[string]any{
		"activity_item_id": cfg.ActivityItemID,
		"quantity":         cfg.Quantity,
		"idempotency_key":  idempotencyKey,
	}
	return doJSON(client, http.MethodPost, path, token, body)
}

func doTrack(client *http.Client, cfg runConfig, idempotencyKey, clientID string) (int, string, string, error) {
	path := fmt.Sprintf("%s/api/v1/seckill/activities/%d/track", strings.TrimRight(cfg.BaseURL, "/"), cfg.ActivityID)
	body := map[string]any{
		"activity_item_id": cfg.ActivityItemID,
		"event_type":       cfg.EventType,
		"client_id":        clientID,
		"idempotency_key":  idempotencyKey,
		"occurred_at_unix": time.Now().Unix(),
	}
	return doJSON(client, http.MethodPost, path, "", body)
}

func doJSON(client *http.Client, method, url, token string, payload any) (int, string, string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, "", "", err
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(raw))
	if err != nil {
		return 0, "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, "", "", err
	}
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return resp.StatusCode, "", "", err
	}
	orderNo := ""
	if len(env.Data) > 0 {
		var order struct {
			OrderNo string `json:"order_no"`
		}
		if err := json.Unmarshal(env.Data, &order); err == nil {
			orderNo = strings.TrimSpace(order.OrderNo)
		}
	}
	return resp.StatusCode, strings.TrimSpace(env.Code), orderNo, nil
}

type summary struct {
	Total         int
	Success       int
	UniqueOrders  int
	NetworkErrors int
	Elapsed       time.Duration
	RPS           float64
	MeanLatency   time.Duration
	P50Latency    time.Duration
	P95Latency    time.Duration
	P99Latency    time.Duration
	MaxLatency    time.Duration
	HTTPStats     map[int]int
	CodeStats     map[string]int
}

func summarize(results []requestResult, elapsed time.Duration) summary {
	s := summary{
		Total:     len(results),
		Elapsed:   elapsed,
		HTTPStats: make(map[int]int),
		CodeStats: make(map[string]int),
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
		}
		if r.Err == nil && r.HTTPStatus == http.StatusOK && r.Code == "OK" {
			s.Success++
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

func assertScenario(cfg runConfig, s summary) error {
	if s.NetworkErrors > cfg.MaxNetworkErrors {
		return fmt.Errorf("network errors exceeded: got=%d max=%d", s.NetworkErrors, cfg.MaxNetworkErrors)
	}
	if cfg.ExpectMaxSuccess >= 0 && s.Success > cfg.ExpectMaxSuccess {
		return fmt.Errorf("success count exceeded: got=%d expect-max=%d", s.Success, cfg.ExpectMaxSuccess)
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

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
	os.Exit(1)
}

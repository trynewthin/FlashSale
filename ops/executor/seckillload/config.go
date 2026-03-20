package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// ─── 常量 ───

const (
	scenarioPurchaseStress = "purchase-stress"
	scenarioIdempotency    = "idempotency"
	scenarioTrackStress    = "track-stress"
	scenarioPurchaseOpen   = "purchase-open"
	scenarioTrackOpen      = "track-open"
	outputText             = "text"
	outputJSON             = "json"
)

// ─── 运行配置 ───

type runConfig struct {
	Scenario         string
	BaseURL          string
	ActivityID       int64
	ActivityItemID   int64
	Concurrency      int
	Requests         int
	OpenRate         int
	OpenDuration     time.Duration
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

// isOpenModel 判断当前是否为开环场景。
func (cfg runConfig) isOpenModel() bool {
	return cfg.Scenario == scenarioPurchaseOpen || cfg.Scenario == scenarioTrackOpen
}

// ─── Flag 解析 ───

func parseFlags() runConfig {
	cfg := runConfig{}
	flag.StringVar(&cfg.Scenario, "scenario", scenarioPurchaseStress, "压测场景: purchase-stress|idempotency|track-stress|purchase-open|track-open")
	flag.StringVar(&cfg.BaseURL, "base-url", envOrDefault("FLASHSALE_BASE_URL", "http://127.0.0.1:8082"), "用户网关地址")
	flag.Int64Var(&cfg.ActivityID, "activity-id", envInt64("FLASHSALE_ACTIVITY_ID", 0), "秒杀活动ID")
	flag.Int64Var(&cfg.ActivityItemID, "item-id", envInt64("FLASHSALE_ACTIVITY_ITEM_ID", 0), "活动商品ID")
	flag.IntVar(&cfg.Concurrency, "concurrency", envInt("FLASHSALE_PRESSURE_CONCURRENCY", 100), "并发数")
	flag.IntVar(&cfg.Requests, "requests", envInt("FLASHSALE_PRESSURE_REQUESTS", 1000), "请求总数")
	flag.IntVar(&cfg.OpenRate, "rate", envInt("FLASHSALE_OPEN_RATE", 0), "开环固定到达率(req/s)")
	flag.DurationVar(&cfg.OpenDuration, "open-duration", envDuration("FLASHSALE_OPEN_DURATION", 30*time.Second), "开环持续时间")
	flag.Int64Var(&cfg.Quantity, "quantity", envInt64("FLASHSALE_PRESSURE_QUANTITY", 1), "购买件数")
	flag.StringVar(&cfg.EventType, "event-type", envOrDefault("FLASHSALE_TRACK_EVENT_TYPE", "pv"), "埋点事件类型")
	flag.StringVar(&cfg.ClientIDPrefix, "client-id-prefix", envOrDefault("FLASHSALE_CLIENT_ID_PREFIX", "perf-client"), "匿名埋点 client_id 前缀")
	flag.StringVar(&cfg.IdempotencyGroup, "idempotency-group", envOrDefault("FLASHSALE_IDEMPOTENCY_GROUP", "same-key-group"), "幂等场景固定分组键")
	flag.IntVar(&cfg.ExpectMaxSuccess, "expect-max-success", envInt("FLASHSALE_EXPECT_MAX_SUCCESS", -1), "断言上限（普通场景按 Success，幂等场景按 Unique Orders；<0 表示不校验）")
	flag.IntVar(&cfg.MaxNetworkErrors, "max-network-errors", envInt("FLASHSALE_MAX_NETWORK_ERRORS", -1), "允许的网络错误上限（<0 表示不限制，仅限真正的网络错误，不含 local_drop）")
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

// ─── 配置校验 ───

func validateConfig(cfg runConfig) error {
	switch cfg.Scenario {
	case scenarioPurchaseStress, scenarioIdempotency, scenarioTrackStress, scenarioPurchaseOpen, scenarioTrackOpen:
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
	if cfg.isOpenModel() {
		if cfg.OpenRate <= 0 {
			return errors.New("开环场景 rate 必须大于 0")
		}
		if cfg.OpenDuration <= 0 {
			return errors.New("开环场景 open-duration 必须大于 0")
		}
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

// ─── Token 加载 ───

func loadTokens(cfg runConfig) ([]string, error) {
	if strings.TrimSpace(cfg.TokenFile) != "" {
		f, err := os.Open(cfg.TokenFile)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		var tokens []string
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			t := strings.TrimSpace(scanner.Text())
			if t != "" {
				tokens = append(tokens, t)
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		if len(tokens) == 0 {
			return nil, fmt.Errorf("token file %s is empty", cfg.TokenFile)
		}
		return tokens, nil
	}
	if strings.TrimSpace(cfg.Token) != "" {
		return []string{cfg.Token}, nil
	}
	return nil, errors.New("token or token-file required (set FLASHSALE_USER_TOKEN or FLASHSALE_TOKENS_FILE)")
}

// ─── HTTP Client ───

func newHTTPClient(cfg runConfig) *http.Client {
	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdlePerHost,
		MaxConnsPerHost:     cfg.MaxConnsPerHost,
		DisableKeepAlives:   cfg.DisableKeepAlive,
		IdleConnTimeout:     90 * time.Second,
	}
	return &http.Client{
		Timeout:   cfg.Timeout,
		Transport: transport,
	}
}

// ─── 环境变量工具 ───

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

// ─── 通用数据结构 ───

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

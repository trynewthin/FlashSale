// perf_preflight 为压测命令提供前置准备与就绪校验，确保错误在压测前暴露。
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"flashsale/cmd/fs/internal/devenv"
)

const (
	perfDefaultTimeout = 3 * time.Second
)

// perfReadinessConfig 定义压测预检需要的关键输入。
type perfReadinessConfig struct {
	Scenario       string
	BaseURL        string
	ActivityID     int64
	ActivityItemID int64
	Quantity       int64
	Token          string
	TokenFile      string
	Timeout        time.Duration
}

type perfAPIEnvelope struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type activityPublicForPreflight struct {
	ActivityID  int64
	StartAtUnix int64
	EndAtUnix   int64
	Status      int32
	Items       []activityItemPublicForPreflight
}

type activityItemPublicForPreflight struct {
	ItemID         int64
	InStock        bool
	MaxQtyPerOrder int64
}

// runPerfPrepare 执行“环境+数据+就绪”准备阶段，不执行压测主流程。
func runPerfPrepare(args []string, envFile string) error {
	fs := flag.NewFlagSet("perf prepare", flag.ContinueOnError)
	skipSmoke := fs.Bool("skip-smoke", false, "跳过 env smoke（mysql/redis/kafka）")
	skipSeed := fs.Bool("skip-seed", false, "跳过 seed-overwrite 数据准备")
	baseURL := fs.String("base-url", "", "覆盖压测网关地址")
	if err := fs.Parse(args); err != nil {
		return err
	}

	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	if strings.TrimSpace(envFile) == "" {
		envFile = "configs/local/dev.env"
	}
	if strings.TrimSpace(envFile) != "-" {
		_ = devenv.Load(devenv.ResolvePath(repoRoot, envFile))
	}
	if strings.TrimSpace(*baseURL) != "" {
		_ = os.Setenv("FLASHSALE_BASE_URL", strings.TrimSpace(*baseURL))
	}
	ensurePerfBaseURL()
	applyPerfDefaultsFromRunlogs(repoRoot)

	fmt.Println("[perf.prepare] 开始准备测试环境")
	if !*skipSmoke {
		fmt.Println("[perf.prepare] step=env.smoke")
		if err := runEnvSmoke([]string{"--env-file=" + envFile}); err != nil {
			return fmt.Errorf("prepare failed at env smoke: %w", err)
		}
	}
	if !*skipSeed {
		userBaseURL, adminBaseURL := deriveSeedGatewayBaseURLs(strings.TrimSpace(os.Getenv("FLASHSALE_BASE_URL")))
		fmt.Printf("[perf.prepare] step=data.seed-overwrite user=%s admin=%s\n", userBaseURL, adminBaseURL)
		if err := runData([]string{
			"seed-overwrite",
			"--force",
			"--env-file=" + envFile,
			"--user-base-url=" + userBaseURL,
			"--admin-base-url=" + adminBaseURL,
		}); err != nil {
			return fmt.Errorf("prepare failed at data seed-overwrite: %w", err)
		}
		applyPerfDefaultsFromRunlogs(repoRoot)
	}
	if err := runPerfPreflightOnce(repoRoot, perfScenarioPurchaseOpen, nil); err != nil {
		return fmt.Errorf("prepare readiness check failed: %w", err)
	}
	fmt.Println("[perf.prepare] ready")
	return nil
}

// runPerfPreflight 在压测主流程前执行就绪校验，必要时支持自动准备后重试一次。
func runPerfPreflight(repoRoot, envFile, scenario string, perfArgs []string, autoPrepare bool) error {
	if err := runPerfPreflightOnce(repoRoot, scenario, perfArgs); err != nil {
		if !autoPrepare {
			return fmt.Errorf("perf preflight failed: %w; 可先执行: fs perf prepare --env-file=%s", err, envFile)
		}
		fmt.Printf("[perf.preflight] 首次失败，尝试自动准备: %v\n", err)
		if prepErr := runPerfPrepare(nil, envFile); prepErr != nil {
			return fmt.Errorf("perf preflight failed: %w; auto-prepare failed: %w", err, prepErr)
		}
		if retryErr := runPerfPreflightOnce(repoRoot, scenario, perfArgs); retryErr != nil {
			return fmt.Errorf("perf preflight retry failed after auto-prepare: %w", retryErr)
		}
	}
	return nil
}

// runPerfPreflightOnce 执行一次就绪检查。
func runPerfPreflightOnce(repoRoot, scenario string, perfArgs []string) error {
	cfg, err := buildPerfReadinessConfig(scenario, perfArgs)
	if err != nil {
		return err
	}
	if cfg.ActivityID <= 0 || cfg.ActivityItemID <= 0 {
		return fmt.Errorf("activity-id 或 item-id 未配置（activity-id=%d item-id=%d）", cfg.ActivityID, cfg.ActivityItemID)
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return fmt.Errorf("base-url 为空")
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = perfDefaultTimeout
	}
	if timeout < 1500*time.Millisecond {
		timeout = 1500 * time.Millisecond
	}

	fmt.Printf("[perf.preflight] 检查网关健康: %s/healthz\n", strings.TrimRight(cfg.BaseURL, "/"))
	if !isGatewayHealthy(cfg.BaseURL, timeout) {
		return fmt.Errorf("gateway health check failed: %s/healthz 不可用", strings.TrimRight(cfg.BaseURL, "/"))
	}

	client := &http.Client{Timeout: timeout}
	activity, err := fetchActivityPublicForPreflight(client, cfg.BaseURL, cfg.ActivityID)
	if err != nil {
		return err
	}
	item, err := findActivityItemForPreflight(activity.Items, cfg.ActivityItemID)
	if err != nil {
		return err
	}
	if err := validateActivityWindowForPreflight(activity); err != nil {
		return err
	}
	if requiresPurchaseToken(cfg.Scenario) {
		if cfg.Quantity <= 0 {
			return fmt.Errorf("quantity 非法: %d", cfg.Quantity)
		}
		if item.MaxQtyPerOrder > 0 && cfg.Quantity > item.MaxQtyPerOrder {
			return fmt.Errorf("quantity=%d 超过 max_qty_per_order=%d", cfg.Quantity, item.MaxQtyPerOrder)
		}
		if !item.InStock {
			return fmt.Errorf("activity_item_id=%d 当前无库存（in_stock=false）", cfg.ActivityItemID)
		}
		token, source, err := resolvePerfToken(cfg.Token, cfg.TokenFile, repoRoot)
		if err != nil {
			return err
		}
		if err := validatePurchaseTokenForPreflight(client, cfg.BaseURL, token); err != nil {
			return fmt.Errorf("token 验证失败（来源 %s）: %w", source, err)
		}
	}
	fmt.Printf("[perf.preflight] ready activity_id=%d item_id=%d scenario=%s\n", cfg.ActivityID, cfg.ActivityItemID, cfg.Scenario)
	return nil
}

// buildPerfReadinessConfig 结合环境变量与命令参数构建预检输入。
func buildPerfReadinessConfig(scenario string, perfArgs []string) (perfReadinessConfig, error) {
	cfg := perfReadinessConfig{
		Scenario:       scenario,
		BaseURL:        strings.TrimSpace(os.Getenv("FLASHSALE_BASE_URL")),
		ActivityID:     envInt64WithFallback("FLASHSALE_ACTIVITY_ID", 0),
		ActivityItemID: envInt64WithFallback("FLASHSALE_ACTIVITY_ITEM_ID", 0),
		Quantity:       envInt64WithFallback("FLASHSALE_PRESSURE_QUANTITY", 1),
		Token:          strings.TrimSpace(os.Getenv("FLASHSALE_USER_TOKEN")),
		TokenFile:      strings.TrimSpace(os.Getenv("FLASHSALE_TOKENS_FILE")),
		Timeout:        envDurationWithFallback("FLASHSALE_HTTP_TIMEOUT", perfDefaultTimeout),
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://127.0.0.1:8082"
	}
	if cfg.Quantity <= 0 {
		cfg.Quantity = 1
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = perfDefaultTimeout
	}

	if value, found, err := findPerfFlagValue(perfArgs, "base-url"); err != nil {
		return perfReadinessConfig{}, err
	} else if found {
		cfg.BaseURL = strings.TrimSpace(value)
	}
	if value, found, err := findPerfFlagValue(perfArgs, "activity-id"); err != nil {
		return perfReadinessConfig{}, err
	} else if found {
		parsed, parseErr := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if parseErr != nil {
			return perfReadinessConfig{}, fmt.Errorf("invalid activity-id: %w", parseErr)
		}
		cfg.ActivityID = parsed
	}
	if value, found, err := findPerfFlagValue(perfArgs, "item-id"); err != nil {
		return perfReadinessConfig{}, err
	} else if found {
		parsed, parseErr := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if parseErr != nil {
			return perfReadinessConfig{}, fmt.Errorf("invalid item-id: %w", parseErr)
		}
		cfg.ActivityItemID = parsed
	}
	if value, found, err := findPerfFlagValue(perfArgs, "quantity"); err != nil {
		return perfReadinessConfig{}, err
	} else if found {
		parsed, parseErr := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if parseErr != nil {
			return perfReadinessConfig{}, fmt.Errorf("invalid quantity: %w", parseErr)
		}
		cfg.Quantity = parsed
	}
	if value, found, err := findPerfFlagValue(perfArgs, "token"); err != nil {
		return perfReadinessConfig{}, err
	} else if found {
		cfg.Token = strings.TrimSpace(value)
	}
	if value, found, err := findPerfFlagValue(perfArgs, "token-file"); err != nil {
		return perfReadinessConfig{}, err
	} else if found {
		cfg.TokenFile = strings.TrimSpace(value)
	}
	if value, found, err := findPerfFlagValue(perfArgs, "timeout"); err != nil {
		return perfReadinessConfig{}, err
	} else if found {
		parsed, parseErr := time.ParseDuration(strings.TrimSpace(value))
		if parseErr != nil {
			return perfReadinessConfig{}, fmt.Errorf("invalid timeout: %w", parseErr)
		}
		cfg.Timeout = parsed
	}
	return cfg, nil
}

// findPerfFlagValue 在参数列表中查找指定 flag（支持 -x y / --x y / -x=y / --x=y）。
func findPerfFlagValue(args []string, flagName string) (string, bool, error) {
	shortName := "-" + strings.TrimSpace(flagName)
	longName := "--" + strings.TrimSpace(flagName)
	found := false
	value := ""
	for index := 0; index < len(args); index++ {
		token := strings.TrimSpace(args[index])
		switch {
		case token == shortName || token == longName:
			if index+1 >= len(args) {
				return "", false, fmt.Errorf("missing value for %s", token)
			}
			next := strings.TrimSpace(args[index+1])
			if next == "" {
				return "", false, fmt.Errorf("empty value for %s", token)
			}
			value = next
			found = true
			index += 1
		case strings.HasPrefix(token, shortName+"="):
			next := strings.TrimSpace(strings.TrimPrefix(token, shortName+"="))
			if next == "" {
				return "", false, fmt.Errorf("empty value for %s", shortName)
			}
			value = next
			found = true
		case strings.HasPrefix(token, longName+"="):
			next := strings.TrimSpace(strings.TrimPrefix(token, longName+"="))
			if next == "" {
				return "", false, fmt.Errorf("empty value for %s", longName)
			}
			value = next
			found = true
		}
	}
	return value, found, nil
}

// fetchActivityPublicForPreflight 拉取活动详情并解析核心字段。
func fetchActivityPublicForPreflight(client *http.Client, baseURL string, activityID int64) (activityPublicForPreflight, error) {
	path := fmt.Sprintf("%s/api/v1/seckill/activities/%d", strings.TrimRight(baseURL, "/"), activityID)
	statusCode, envelope, err := callPerfAPIEnvelope(client, http.MethodGet, path, "", nil)
	if err != nil {
		return activityPublicForPreflight{}, fmt.Errorf("query activity failed: %w", err)
	}
	if statusCode < 200 || statusCode >= 300 {
		return activityPublicForPreflight{}, fmt.Errorf("query activity failed: http=%d code=%s message=%s", statusCode, envelope.Code, envelope.Message)
	}
	if strings.TrimSpace(envelope.Code) != "OK" {
		return activityPublicForPreflight{}, fmt.Errorf("query activity failed: code=%s message=%s", envelope.Code, envelope.Message)
	}

	var payload struct {
		Activity struct {
			ActivityID  json.Number `json:"activity_id"`
			StartAtUnix json.Number `json:"start_at_unix"`
			EndAtUnix   json.Number `json:"end_at_unix"`
			Status      int32       `json:"status"`
			Items       []struct {
				ItemID         json.Number `json:"item_id"`
				InStock        bool        `json:"in_stock"`
				MaxQtyPerOrder json.Number `json:"max_qty_per_order"`
			} `json:"items"`
		} `json:"activity"`
	}
	decoder := json.NewDecoder(bytes.NewReader(envelope.Data))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return activityPublicForPreflight{}, fmt.Errorf("decode activity payload failed: %w", err)
	}
	if payload.Activity.ActivityID == "" {
		return activityPublicForPreflight{}, fmt.Errorf("activity payload missing activity_id")
	}

	result := activityPublicForPreflight{Status: payload.Activity.Status}
	result.ActivityID, err = parseJSONNumberField(payload.Activity.ActivityID, "activity_id")
	if err != nil {
		return activityPublicForPreflight{}, err
	}
	if payload.Activity.StartAtUnix != "" {
		result.StartAtUnix, err = parseJSONNumberField(payload.Activity.StartAtUnix, "start_at_unix")
		if err != nil {
			return activityPublicForPreflight{}, err
		}
	}
	if payload.Activity.EndAtUnix != "" {
		result.EndAtUnix, err = parseJSONNumberField(payload.Activity.EndAtUnix, "end_at_unix")
		if err != nil {
			return activityPublicForPreflight{}, err
		}
	}
	result.Items = make([]activityItemPublicForPreflight, 0, len(payload.Activity.Items))
	for _, item := range payload.Activity.Items {
		if item.ItemID == "" {
			continue
		}
		itemID, parseErr := parseJSONNumberField(item.ItemID, "item_id")
		if parseErr != nil {
			return activityPublicForPreflight{}, parseErr
		}
		maxQty := int64(0)
		if item.MaxQtyPerOrder != "" {
			maxQty, parseErr = parseJSONNumberField(item.MaxQtyPerOrder, "max_qty_per_order")
			if parseErr != nil {
				return activityPublicForPreflight{}, parseErr
			}
		}
		result.Items = append(result.Items, activityItemPublicForPreflight{
			ItemID:         itemID,
			InStock:        item.InStock,
			MaxQtyPerOrder: maxQty,
		})
	}
	return result, nil
}

// callPerfAPIEnvelope 调用网关 API 并按统一响应包体解码。
func callPerfAPIEnvelope(client *http.Client, method, requestURL, token string, body any) (int, perfAPIEnvelope, error) {
	return callPerfAPIEnvelopeWithHeaders(client, method, requestURL, token, body, nil)
}

// callPerfAPIEnvelopeWithHeaders 调用网关 API 并按统一响应包体解码，支持附加请求头。
func callPerfAPIEnvelopeWithHeaders(
	client *http.Client,
	method,
	requestURL,
	token string,
	body any,
	extraHeaders map[string]string,
) (int, perfAPIEnvelope, error) {
	var payloadReader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, perfAPIEnvelope{}, err
		}
		payloadReader = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, requestURL, payloadReader)
	if err != nil {
		return 0, perfAPIEnvelope{}, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}
	for key, value := range extraHeaders {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" || trimmedValue == "" {
			continue
		}
		req.Header.Set(trimmedKey, trimmedValue)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, perfAPIEnvelope{}, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, perfAPIEnvelope{}, err
	}
	var envelope perfAPIEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return resp.StatusCode, perfAPIEnvelope{}, fmt.Errorf("decode envelope failed (http=%d): %w", resp.StatusCode, err)
	}
	return resp.StatusCode, envelope, nil
}

// validateActivityWindowForPreflight 校验活动状态与时间窗口。
func validateActivityWindowForPreflight(activity activityPublicForPreflight) error {
	if activity.Status != 1 {
		return fmt.Errorf("activity_id=%d 非发布状态（status=%d）", activity.ActivityID, activity.Status)
	}
	nowUnix := time.Now().Unix()
	if activity.StartAtUnix > 0 && nowUnix < activity.StartAtUnix {
		return fmt.Errorf("activity_id=%d 未开始（start_at=%s）", activity.ActivityID, time.Unix(activity.StartAtUnix, 0).Format(time.RFC3339))
	}
	if activity.EndAtUnix > 0 && nowUnix > activity.EndAtUnix {
		return fmt.Errorf("activity_id=%d 已结束（end_at=%s）", activity.ActivityID, time.Unix(activity.EndAtUnix, 0).Format(time.RFC3339))
	}
	return nil
}

// findActivityItemForPreflight 确认活动内存在指定 item。
func findActivityItemForPreflight(items []activityItemPublicForPreflight, itemID int64) (activityItemPublicForPreflight, error) {
	for _, item := range items {
		if item.ItemID == itemID {
			return item, nil
		}
	}
	return activityItemPublicForPreflight{}, fmt.Errorf("activity_item_id=%d 不存在于当前活动", itemID)
}

// validatePurchaseTokenForPreflight 校验购买场景令牌可用性。
func validatePurchaseTokenForPreflight(client *http.Client, baseURL, token string) error {
	path := fmt.Sprintf("%s/api/v1/orders?page=1&page_size=1", strings.TrimRight(baseURL, "/"))
	statusCode, envelope, err := callPerfAPIEnvelope(client, http.MethodGet, path, token, nil)
	if err != nil {
		return err
	}
	if statusCode < 200 || statusCode >= 300 || strings.TrimSpace(envelope.Code) != "OK" {
		return fmt.Errorf("http=%d code=%s message=%s", statusCode, envelope.Code, envelope.Message)
	}
	return nil
}

// resolvePerfToken 获取购买场景校验所需 token（优先直接 token，再取 token-file 第一条）。
func resolvePerfToken(token, tokenFile, repoRoot string) (string, string, error) {
	if v := strings.TrimSpace(token); v != "" {
		return v, "token", nil
	}
	if strings.TrimSpace(tokenFile) == "" {
		return "", "", fmt.Errorf("购买场景缺少 token/token-file")
	}
	path := strings.TrimSpace(tokenFile)
	if !filepath.IsAbs(path) {
		path = filepath.Join(repoRoot, path)
	}
	file, err := os.Open(path)
	if err != nil {
		return "", "", fmt.Errorf("open token-file failed: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return line, path, nil
	}
	if err := scanner.Err(); err != nil {
		return "", "", fmt.Errorf("read token-file failed: %w", err)
	}
	return "", "", fmt.Errorf("token-file is empty: %s", path)
}

// deriveSeedGatewayBaseURLs 推导 seed-overwrite 所需的 user/admin 网关地址。
func deriveSeedGatewayBaseURLs(baseURL string) (string, string) {
	userBase := strings.TrimSpace(os.Getenv("FLASHSALE_USER_BASE_URL"))
	adminBase := strings.TrimSpace(os.Getenv("FLASHSALE_ADMIN_BASE_URL"))
	if userBase != "" && adminBase != "" {
		return strings.TrimRight(userBase, "/"), strings.TrimRight(adminBase, "/")
	}
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = "http://127.0.0.1:8082"
	}
	if userBase == "" {
		userBase = base
	}
	if adminBase == "" {
		adminBase = base
		parsed, err := url.Parse(base)
		if err == nil && parsed.Port() == "8082" {
			hostname := parsed.Hostname()
			if hostname != "" {
				parsed.Host = net.JoinHostPort(hostname, "8083")
				adminBase = parsed.String()
			}
		}
	}
	return strings.TrimRight(userBase, "/"), strings.TrimRight(adminBase, "/")
}

// parseJSONNumberField 将 json.Number 转为 int64，并携带字段名。
func parseJSONNumberField(value json.Number, field string) (int64, error) {
	parsed, err := value.Int64()
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", field, err)
	}
	return parsed, nil
}

// requiresPurchaseToken 判断场景是否必须校验用户 token。
func requiresPurchaseToken(scenario string) bool {
	switch scenario {
	case perfScenarioPurchaseStress, perfScenarioIdempotency, perfScenarioPurchaseOpen:
		return true
	default:
		return false
	}
}

func envInt64WithFallback(key string, fallback int64) int64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDurationWithFallback(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

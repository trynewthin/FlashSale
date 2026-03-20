package perf

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

	"flashsale/ops/backend/shared/devenv"
)

const DefaultTimeout = 3 * time.Second

type ReadinessConfig struct {
	Scenario       string
	BaseURL        string
	ActivityID     int64
	ActivityItemID int64
	Quantity       int64
	Token          string
	TokenFile      string
	Timeout        time.Duration
}

type APIEnvelope struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type ActivityPublicForPreflight struct {
	ActivityID  int64
	StartAtUnix int64
	EndAtUnix   int64
	Status      int32
	Items       []ActivityItemPublicForPreflight
}

type ActivityItemPublicForPreflight struct {
	ItemID         int64
	InStock        bool
	MaxQtyPerOrder int64
}

type PrepareDeps struct {
	RunSmoke         func(envFile string) error
	RunSeedOverwrite func(envFile, userBaseURL, adminBaseURL string) error
}

type PrepareFunc func(args []string, envFile string) error

func RunPrepare(args []string, envFile string, deps PrepareDeps) error {
	fs := flag.NewFlagSet("perf prepare", flag.ContinueOnError)
	skipSmoke := fs.Bool("skip-smoke", false, "skip env smoke")
	skipSeed := fs.Bool("skip-seed", false, "skip seed-overwrite")
	baseURL := fs.String("base-url", "", "override gateway base url")
	if err := fs.Parse(args); err != nil {
		return err
	}

	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	if strings.TrimSpace(envFile) == "" {
		envFile = "configs/deploy.env"
	}
	if strings.TrimSpace(envFile) != "-" {
		_ = devenv.Load(devenv.ResolvePath(repoRoot, envFile))
	}
	if strings.TrimSpace(*baseURL) != "" {
		_ = os.Setenv("FLASHSALE_BASE_URL", strings.TrimSpace(*baseURL))
	}
	EnsureBaseURL()
	ApplyDefaultsFromRunlogs(repoRoot)

	fmt.Println("[perf.prepare] start")
	if !*skipSmoke {
		if deps.RunSmoke == nil {
			return fmt.Errorf("prepare smoke dependency is not configured")
		}
		fmt.Println("[perf.prepare] step=env.smoke")
		if err := deps.RunSmoke(envFile); err != nil {
			return fmt.Errorf("prepare failed at env smoke: %w", err)
		}
	}
	if !*skipSeed {
		if deps.RunSeedOverwrite == nil {
			return fmt.Errorf("prepare seed dependency is not configured")
		}
		userBaseURL, adminBaseURL := DeriveSeedGatewayBaseURLs(strings.TrimSpace(os.Getenv("FLASHSALE_BASE_URL")))
		fmt.Printf("[perf.prepare] step=data.seed-overwrite user=%s admin=%s\n", userBaseURL, adminBaseURL)
		if err := deps.RunSeedOverwrite(envFile, userBaseURL, adminBaseURL); err != nil {
			return fmt.Errorf("prepare failed at data seed-overwrite: %w", err)
		}
		ApplyDefaultsFromRunlogs(repoRoot)
	}
	if err := RunPreflightOnce(repoRoot, ScenarioPurchaseOpen, nil); err != nil {
		return fmt.Errorf("prepare readiness check failed: %w", err)
	}
	fmt.Println("[perf.prepare] ready")
	return nil
}

func RunPreflight(repoRoot, envFile, scenario string, perfArgs []string, autoPrepare bool, runPrepare PrepareFunc) error {
	if err := RunPreflightOnce(repoRoot, scenario, perfArgs); err != nil {
		if !autoPrepare {
			return fmt.Errorf("perf preflight failed: %w; run fs perf prepare --env-file=%s first", err, envFile)
		}
		if runPrepare == nil {
			return fmt.Errorf("perf preflight failed: %w; auto-prepare is unavailable", err)
		}
		fmt.Printf("[perf.preflight] first attempt failed, auto-prepare: %v\n", err)
		if prepErr := runPrepare(nil, envFile); prepErr != nil {
			return fmt.Errorf("perf preflight failed: %w; auto-prepare failed: %w", err, prepErr)
		}
		if retryErr := RunPreflightOnce(repoRoot, scenario, perfArgs); retryErr != nil {
			return fmt.Errorf("perf preflight retry failed after auto-prepare: %w", retryErr)
		}
	}
	return nil
}

func RunPreflightOnce(repoRoot, scenario string, perfArgs []string) error {
	cfg, err := BuildReadinessConfig(scenario, perfArgs)
	if err != nil {
		return err
	}
	if cfg.ActivityID <= 0 || cfg.ActivityItemID <= 0 {
		return fmt.Errorf("activity-id or item-id is missing (activity-id=%d item-id=%d)", cfg.ActivityID, cfg.ActivityItemID)
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return fmt.Errorf("base-url is empty")
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	if timeout < 1500*time.Millisecond {
		timeout = 1500 * time.Millisecond
	}

	fmt.Printf("[perf.preflight] checking gateway %s/healthz\n", strings.TrimRight(cfg.BaseURL, "/"))
	if !IsGatewayHealthy(cfg.BaseURL, timeout) {
		return fmt.Errorf("gateway health check failed: %s/healthz is unavailable", strings.TrimRight(cfg.BaseURL, "/"))
	}

	client := &http.Client{Timeout: timeout}
	activity, err := FetchActivityPublicForPreflight(client, cfg.BaseURL, cfg.ActivityID)
	if err != nil {
		return err
	}
	item, err := FindActivityItemForPreflight(activity.Items, cfg.ActivityItemID)
	if err != nil {
		return err
	}
	if err := ValidateActivityWindowForPreflight(activity); err != nil {
		return err
	}
	if RequiresPurchaseToken(cfg.Scenario) {
		if cfg.Quantity <= 0 {
			return fmt.Errorf("invalid quantity: %d", cfg.Quantity)
		}
		if item.MaxQtyPerOrder > 0 && cfg.Quantity > item.MaxQtyPerOrder {
			return fmt.Errorf("quantity=%d exceeds max_qty_per_order=%d", cfg.Quantity, item.MaxQtyPerOrder)
		}
		if !item.InStock {
			return fmt.Errorf("activity_item_id=%d is out of stock", cfg.ActivityItemID)
		}
		token, source, err := ResolveToken(cfg.Token, cfg.TokenFile, repoRoot)
		if err != nil {
			return err
		}
		if err := ValidatePurchaseTokenForPreflight(client, cfg.BaseURL, token); err != nil {
			return fmt.Errorf("token validation failed (%s): %w", source, err)
		}
	}
	fmt.Printf("[perf.preflight] ready activity_id=%d item_id=%d scenario=%s\n", cfg.ActivityID, cfg.ActivityItemID, cfg.Scenario)
	return nil
}

func BuildReadinessConfig(scenario string, perfArgs []string) (ReadinessConfig, error) {
	cfg := ReadinessConfig{
		Scenario:       scenario,
		BaseURL:        strings.TrimSpace(os.Getenv("FLASHSALE_BASE_URL")),
		ActivityID:     EnvInt64WithFallback("FLASHSALE_ACTIVITY_ID", 0),
		ActivityItemID: EnvInt64WithFallback("FLASHSALE_ACTIVITY_ITEM_ID", 0),
		Quantity:       EnvInt64WithFallback("FLASHSALE_PRESSURE_QUANTITY", 1),
		Token:          strings.TrimSpace(os.Getenv("FLASHSALE_USER_TOKEN")),
		TokenFile:      strings.TrimSpace(os.Getenv("FLASHSALE_TOKENS_FILE")),
		Timeout:        EnvDurationWithFallback("FLASHSALE_HTTP_TIMEOUT", DefaultTimeout),
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = strings.TrimSpace(os.Getenv("FLASHSALE_USER_BASE_URL"))
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://127.0.0.1:8082"
	}
	if cfg.Quantity <= 0 {
		cfg.Quantity = 1
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeout
	}

	if value, found, err := FindFlagValue(perfArgs, "base-url"); err != nil {
		return ReadinessConfig{}, err
	} else if found {
		cfg.BaseURL = strings.TrimSpace(value)
	}
	if value, found, err := FindFlagValue(perfArgs, "activity-id"); err != nil {
		return ReadinessConfig{}, err
	} else if found {
		parsed, parseErr := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if parseErr != nil {
			return ReadinessConfig{}, fmt.Errorf("invalid activity-id: %w", parseErr)
		}
		cfg.ActivityID = parsed
	}
	if value, found, err := FindFlagValue(perfArgs, "item-id"); err != nil {
		return ReadinessConfig{}, err
	} else if found {
		parsed, parseErr := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if parseErr != nil {
			return ReadinessConfig{}, fmt.Errorf("invalid item-id: %w", parseErr)
		}
		cfg.ActivityItemID = parsed
	}
	if value, found, err := FindFlagValue(perfArgs, "quantity"); err != nil {
		return ReadinessConfig{}, err
	} else if found {
		parsed, parseErr := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if parseErr != nil {
			return ReadinessConfig{}, fmt.Errorf("invalid quantity: %w", parseErr)
		}
		cfg.Quantity = parsed
	}
	if value, found, err := FindFlagValue(perfArgs, "token"); err != nil {
		return ReadinessConfig{}, err
	} else if found {
		cfg.Token = strings.TrimSpace(value)
	}
	if value, found, err := FindFlagValue(perfArgs, "token-file"); err != nil {
		return ReadinessConfig{}, err
	} else if found {
		cfg.TokenFile = strings.TrimSpace(value)
	}
	if value, found, err := FindFlagValue(perfArgs, "timeout"); err != nil {
		return ReadinessConfig{}, err
	} else if found {
		parsed, parseErr := time.ParseDuration(strings.TrimSpace(value))
		if parseErr != nil {
			return ReadinessConfig{}, fmt.Errorf("invalid timeout: %w", parseErr)
		}
		cfg.Timeout = parsed
	}
	return cfg, nil
}

func FindFlagValue(args []string, flagName string) (string, bool, error) {
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
			index++
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

func FetchActivityPublicForPreflight(client *http.Client, baseURL string, activityID int64) (ActivityPublicForPreflight, error) {
	requestURL := fmt.Sprintf("%s/api/v1/seckill/activities/%d", strings.TrimRight(baseURL, "/"), activityID)
	statusCode, envelope, err := CallAPIEnvelope(client, http.MethodGet, requestURL, "", nil)
	if err != nil {
		return ActivityPublicForPreflight{}, fmt.Errorf("query activity failed: %w", err)
	}
	if statusCode < 200 || statusCode >= 300 {
		return ActivityPublicForPreflight{}, fmt.Errorf("query activity failed: http=%d code=%s message=%s", statusCode, envelope.Code, envelope.Message)
	}
	if strings.TrimSpace(envelope.Code) != "OK" {
		return ActivityPublicForPreflight{}, fmt.Errorf("query activity failed: code=%s message=%s", envelope.Code, envelope.Message)
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
		return ActivityPublicForPreflight{}, fmt.Errorf("decode activity payload failed: %w", err)
	}
	if payload.Activity.ActivityID == "" {
		return ActivityPublicForPreflight{}, fmt.Errorf("activity payload missing activity_id")
	}

	result := ActivityPublicForPreflight{Status: payload.Activity.Status}
	result.ActivityID, err = ParseJSONNumberField(payload.Activity.ActivityID, "activity_id")
	if err != nil {
		return ActivityPublicForPreflight{}, err
	}
	if payload.Activity.StartAtUnix != "" {
		result.StartAtUnix, err = ParseJSONNumberField(payload.Activity.StartAtUnix, "start_at_unix")
		if err != nil {
			return ActivityPublicForPreflight{}, err
		}
	}
	if payload.Activity.EndAtUnix != "" {
		result.EndAtUnix, err = ParseJSONNumberField(payload.Activity.EndAtUnix, "end_at_unix")
		if err != nil {
			return ActivityPublicForPreflight{}, err
		}
	}
	result.Items = make([]ActivityItemPublicForPreflight, 0, len(payload.Activity.Items))
	for _, item := range payload.Activity.Items {
		if item.ItemID == "" {
			continue
		}
		itemID, parseErr := ParseJSONNumberField(item.ItemID, "item_id")
		if parseErr != nil {
			return ActivityPublicForPreflight{}, parseErr
		}
		maxQty := int64(0)
		if item.MaxQtyPerOrder != "" {
			maxQty, parseErr = ParseJSONNumberField(item.MaxQtyPerOrder, "max_qty_per_order")
			if parseErr != nil {
				return ActivityPublicForPreflight{}, parseErr
			}
		}
		result.Items = append(result.Items, ActivityItemPublicForPreflight{
			ItemID:         itemID,
			InStock:        item.InStock,
			MaxQtyPerOrder: maxQty,
		})
	}
	return result, nil
}

func CallAPIEnvelope(client *http.Client, method, requestURL, token string, body any) (int, APIEnvelope, error) {
	return CallAPIEnvelopeWithHeaders(client, method, requestURL, token, body, nil)
}

func CallAPIEnvelopeWithHeaders(client *http.Client, method, requestURL, token string, body any, extraHeaders map[string]string) (int, APIEnvelope, error) {
	var payloadReader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, APIEnvelope{}, err
		}
		payloadReader = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, requestURL, payloadReader)
	if err != nil {
		return 0, APIEnvelope{}, err
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
		return 0, APIEnvelope{}, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, APIEnvelope{}, err
	}
	var envelope APIEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return resp.StatusCode, APIEnvelope{}, fmt.Errorf("decode envelope failed (http=%d): %w", resp.StatusCode, err)
	}
	return resp.StatusCode, envelope, nil
}

func ValidateActivityWindowForPreflight(activity ActivityPublicForPreflight) error {
	if activity.Status != 1 {
		return fmt.Errorf("activity_id=%d is not published (status=%d)", activity.ActivityID, activity.Status)
	}
	nowUnix := time.Now().Unix()
	if activity.StartAtUnix > 0 && nowUnix < activity.StartAtUnix {
		return fmt.Errorf("activity_id=%d has not started yet (start_at=%s)", activity.ActivityID, time.Unix(activity.StartAtUnix, 0).Format(time.RFC3339))
	}
	if activity.EndAtUnix > 0 && nowUnix > activity.EndAtUnix {
		return fmt.Errorf("activity_id=%d has already ended (end_at=%s)", activity.ActivityID, time.Unix(activity.EndAtUnix, 0).Format(time.RFC3339))
	}
	return nil
}

func FindActivityItemForPreflight(items []ActivityItemPublicForPreflight, itemID int64) (ActivityItemPublicForPreflight, error) {
	for _, item := range items {
		if item.ItemID == itemID {
			return item, nil
		}
	}
	return ActivityItemPublicForPreflight{}, fmt.Errorf("activity_item_id=%d does not exist in the activity", itemID)
}

func ValidatePurchaseTokenForPreflight(client *http.Client, baseURL, token string) error {
	requestURL := fmt.Sprintf("%s/api/v1/orders?page=1&page_size=1", strings.TrimRight(baseURL, "/"))
	statusCode, envelope, err := CallAPIEnvelope(client, http.MethodGet, requestURL, token, nil)
	if err != nil {
		return err
	}
	if statusCode < 200 || statusCode >= 300 || strings.TrimSpace(envelope.Code) != "OK" {
		return fmt.Errorf("http=%d code=%s message=%s", statusCode, envelope.Code, envelope.Message)
	}
	return nil
}

func ResolveToken(token, tokenFile, repoRoot string) (string, string, error) {
	if value := strings.TrimSpace(token); value != "" {
		return value, "token", nil
	}
	if strings.TrimSpace(tokenFile) == "" {
		return "", "", fmt.Errorf("purchase scenarios require token or token-file")
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

func DeriveSeedGatewayBaseURLs(baseURL string) (string, string) {
	userBaseURL := strings.TrimSpace(os.Getenv("FLASHSALE_USER_BASE_URL"))
	adminBaseURL := strings.TrimSpace(os.Getenv("FLASHSALE_ADMIN_BASE_URL"))
	if userBaseURL != "" && adminBaseURL != "" {
		return strings.TrimRight(userBaseURL, "/"), strings.TrimRight(adminBaseURL, "/")
	}
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = "http://127.0.0.1:8082"
	}
	if userBaseURL == "" {
		userBaseURL = base
	}
	if adminBaseURL == "" {
		adminBaseURL = base
		parsed, err := url.Parse(base)
		if err == nil && parsed.Port() == "8082" {
			hostname := parsed.Hostname()
			if hostname != "" {
				parsed.Host = net.JoinHostPort(hostname, "8083")
				adminBaseURL = parsed.String()
			}
		}
	}
	return strings.TrimRight(userBaseURL, "/"), strings.TrimRight(adminBaseURL, "/")
}

func ParseJSONNumberField(value json.Number, field string) (int64, error) {
	parsed, err := value.Int64()
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", field, err)
	}
	return parsed, nil
}

func RequiresPurchaseToken(scenario string) bool {
	switch scenario {
	case ScenarioPurchaseStress, ScenarioIdempotency, ScenarioPurchaseOpen:
		return true
	default:
		return false
	}
}

func EnvInt64WithFallback(key string, fallback int64) int64 {
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

func EnvDurationWithFallback(key string, fallback time.Duration) time.Duration {
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

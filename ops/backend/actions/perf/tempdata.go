package perf

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type TempData struct {
	BaseURL      string
	AdminBaseURL string
	UserBaseURL  string

	AdminToken   string
	ProductID    int64
	ActivityID   int64
	ItemID       int64
	UserIDs      []int64
	TokenFile    string
	UniqueSuffix string
	UsedDBPath   bool
}

func SetupTempData(repoRoot, scenario string, perfArgs []string, options ControlOptions) (*TempData, error) {
	var finalErr error

	cfg, err := BuildReadinessConfig(scenario, perfArgs)
	if err != nil {
		return nil, fmt.Errorf("build readiness config failed: %w", err)
	}
	userBaseURL, adminBaseURL := DeriveSeedGatewayBaseURLs(cfg.BaseURL)
	if strings.TrimSpace(userBaseURL) == "" || strings.TrimSpace(adminBaseURL) == "" {
		return nil, fmt.Errorf("derive gateway base urls failed")
	}

	adminToken, err := LoginAdmin(adminBaseURL, repoRoot)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	temp := &TempData{
		BaseURL:      cfg.BaseURL,
		AdminBaseURL: adminBaseURL,
		UserBaseURL:  userBaseURL,
		AdminToken:   adminToken,
	}
	defer func() {
		if finalErr == nil {
			return
		}
		if cleanupErr := CleanupTempData(temp); cleanupErr != nil {
			fmt.Fprintf(os.Stderr, "WARN: rollback temp data failed: %v\n", cleanupErr)
		}
	}()

	uniqueSuffix := strconv.FormatInt(now.UnixNano(), 10)

	productID, err := CreateTempProduct(adminBaseURL, adminToken, uniqueSuffix)
	if err != nil {
		finalErr = err
		return nil, err
	}
	temp.ProductID = productID

	activityID, err := CreateTempActivity(adminBaseURL, adminToken, now, uniqueSuffix)
	if err != nil {
		finalErr = err
		return nil, err
	}
	temp.ActivityID = activityID

	itemID, err := CreateTempActivityItem(adminBaseURL, adminToken, activityID, productID)
	if err != nil {
		finalErr = err
		return nil, err
	}
	temp.ItemID = itemID

	if err := PublishTempActivity(adminBaseURL, adminToken, activityID); err != nil {
		finalErr = err
		return nil, err
	}

	needUsers := 0
	if RequiresPurchaseToken(scenario) {
		needUsers = ResolveTempUserCount(scenario, options.TempUsers, perfArgs)
	}
	if needUsers > 0 {
		temp.UniqueSuffix = uniqueSuffix
		userIDs, tokenFile, dbErr := createTempUsersDirectDB(repoRoot, needUsers, uniqueSuffix)
		if dbErr == nil {
			temp.UserIDs = userIDs
			temp.TokenFile = tokenFile
			temp.UsedDBPath = true
		} else {
			fmt.Printf("[perf.temp] db direct insert failed (%v), falling back to HTTP\n", dbErr)
			userIDs, tokenFile, err := CreateTempUsersAndTokens(userBaseURL, repoRoot, needUsers, uniqueSuffix)
			if err != nil {
				finalErr = err
				return nil, err
			}
			temp.UserIDs = userIDs
			temp.TokenFile = tokenFile
		}
		_ = os.Unsetenv("FLASHSALE_USER_TOKEN")
		_ = os.Setenv("FLASHSALE_TOKENS_FILE", temp.TokenFile)
	}

	fmt.Printf("[perf.temp] ready product_id=%d activity_id=%d item_id=%d users=%d\n", temp.ProductID, temp.ActivityID, temp.ItemID, len(temp.UserIDs))
	return temp, nil
}

func CleanupTempData(temp *TempData) error {
	if temp == nil {
		return nil
	}

	errs := make([]string, 0, 4)
	if temp.ActivityID > 0 {
		if err := DeleteTempActivity(temp.AdminBaseURL, temp.AdminToken, temp.ActivityID); err != nil {
			errs = append(errs, "delete activity: "+err.Error())
		}
	}
	if temp.ProductID > 0 {
		if err := DeleteTempProduct(temp.AdminBaseURL, temp.AdminToken, temp.ProductID); err != nil {
			errs = append(errs, "delete product: "+err.Error())
		}
	}
	if len(temp.UserIDs) > 0 {
		if temp.UsedDBPath {
			if err := cleanupTempUsersDirectDB(temp.UserIDs); err != nil {
				errs = append(errs, "db cleanup users: "+err.Error())
			}
		} else {
			for _, userID := range temp.UserIDs {
				if userID <= 0 {
					continue
				}
				if err := DeleteTempUser(temp.AdminBaseURL, temp.AdminToken, userID); err != nil {
					errs = append(errs, fmt.Sprintf("delete user %d: %s", userID, err.Error()))
				}
			}
		}
	}
	if temp.TokenFile != "" {
		if err := os.Remove(temp.TokenFile); err != nil && !os.IsNotExist(err) {
			errs = append(errs, "remove token file: "+err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	fmt.Printf("[perf.temp] cleaned activity_id=%d product_id=%d users=%d\n", temp.ActivityID, temp.ProductID, len(temp.UserIDs))
	return nil
}

func LoginAdmin(adminBaseURL, repoRoot string) (string, error) {
	type credential struct {
		Username string
		Password string
	}

	candidates := make([]credential, 0, 4)
	add := func(username, password string) {
		username = strings.TrimSpace(username)
		password = strings.TrimSpace(password)
		if username == "" || password == "" {
			return
		}
		candidates = append(candidates, credential{Username: username, Password: password})
	}

	add(os.Getenv("FLASHSALE_PERF_ADMIN_USERNAME"), os.Getenv("FLASHSALE_PERF_ADMIN_PASSWORD"))
	add(os.Getenv("FLASHSALE_ADMIN_USERNAME"), os.Getenv("FLASHSALE_ADMIN_PASSWORD"))

	seedResultPath := filepath.Join(repoRoot, "log", "data", "seed-overwrite.result.json")
	if raw, err := os.ReadFile(seedResultPath); err == nil {
		var parsed struct {
			Admin struct {
				Username string `json:"username"`
				Password string `json:"password"`
			} `json:"admin"`
		}
		if json.Unmarshal(raw, &parsed) == nil {
			add(parsed.Admin.Username, parsed.Admin.Password)
		}
	}
	add("admin_root", "Admin12345")

	errs := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		token, err := AdminLoginOnce(adminBaseURL, candidate.Username, candidate.Password)
		if err == nil {
			return token, nil
		}
		errs = append(errs, fmt.Sprintf("%s: %v", candidate.Username, err))
	}
	return "", fmt.Errorf("admin login failed for all candidates; set FLASHSALE_PERF_ADMIN_USERNAME/FLASHSALE_PERF_ADMIN_PASSWORD; details: %s", strings.Join(errs, " | "))
}

func AdminLoginOnce(adminBaseURL, username, password string) (string, error) {
	requestURL := strings.TrimRight(adminBaseURL, "/") + "/api/v1/admin/auth/login"
	statusCode, envelope, err := CallAPIEnvelope(DefaultHTTPClient(), "POST", requestURL, "", map[string]any{
		"username": username,
		"password": password,
	})
	if err != nil {
		return "", err
	}
	if statusCode < 200 || statusCode >= 300 || strings.TrimSpace(envelope.Code) != "OK" {
		return "", fmt.Errorf("http=%d code=%s message=%s", statusCode, envelope.Code, envelope.Message)
	}

	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(envelope.Data, &payload); err != nil {
		return "", err
	}
	if strings.TrimSpace(payload.AccessToken) == "" {
		return "", fmt.Errorf("empty access_token")
	}
	return strings.TrimSpace(payload.AccessToken), nil
}

func CreateTempProduct(adminBaseURL, adminToken, suffix string) (int64, error) {
	requestURL := strings.TrimRight(adminBaseURL, "/") + "/api/v1/admin/products"
	statusCode, envelope, err := CallAPIEnvelope(DefaultHTTPClient(), "POST", requestURL, adminToken, map[string]any{
		"name":        "perf-temp-product-" + suffix,
		"main_image":  "https://example.com/perf-temp.png",
		"description": "perf temporary product",
		"price_cent":  9900,
		"stock":       200000,
		"status":      1,
	})
	if err != nil {
		return 0, err
	}
	if statusCode < 200 || statusCode >= 300 || strings.TrimSpace(envelope.Code) != "OK" {
		return 0, fmt.Errorf("create product failed: http=%d code=%s message=%s", statusCode, envelope.Code, envelope.Message)
	}

	var payload struct {
		Product struct {
			ProductID json.Number `json:"product_id"`
		} `json:"product"`
	}
	if err := json.Unmarshal(envelope.Data, &payload); err != nil {
		return 0, err
	}
	return ParseJSONNumberField(payload.Product.ProductID, "product_id")
}

func DeleteTempProduct(adminBaseURL, adminToken string, productID int64) error {
	requestURL := fmt.Sprintf("%s/api/v1/admin/products/%d", strings.TrimRight(adminBaseURL, "/"), productID)
	statusCode, envelope, err := CallAPIEnvelope(DefaultHTTPClient(), "DELETE", requestURL, adminToken, nil)
	if err != nil {
		return err
	}
	if statusCode < 200 || statusCode >= 300 || strings.TrimSpace(envelope.Code) != "OK" {
		return fmt.Errorf("http=%d code=%s message=%s", statusCode, envelope.Code, envelope.Message)
	}
	return nil
}

func CreateTempActivity(adminBaseURL, adminToken string, now time.Time, suffix string) (int64, error) {
	requestURL := strings.TrimRight(adminBaseURL, "/") + "/api/v1/admin/seckill/activities"
	statusCode, envelope, err := CallAPIEnvelope(DefaultHTTPClient(), "POST", requestURL, adminToken, map[string]any{
		"title":             "perf-temp-activity-" + suffix,
		"description":       "perf temporary activity",
		"style_config_json": "{}",
		"start_at_unix":     now.Add(-30 * time.Second).Unix(),
		"end_at_unix":       now.Add(20 * time.Minute).Unix(),
	})
	if err != nil {
		return 0, err
	}
	if statusCode < 200 || statusCode >= 300 || strings.TrimSpace(envelope.Code) != "OK" {
		return 0, fmt.Errorf("create activity failed: http=%d code=%s message=%s", statusCode, envelope.Code, envelope.Message)
	}

	var payload struct {
		Activity struct {
			ActivityID json.Number `json:"activity_id"`
		} `json:"activity"`
	}
	if err := json.Unmarshal(envelope.Data, &payload); err != nil {
		return 0, err
	}
	return ParseJSONNumberField(payload.Activity.ActivityID, "activity_id")
}

func DeleteTempActivity(adminBaseURL, adminToken string, activityID int64) error {
	requestURL := fmt.Sprintf("%s/api/v1/admin/seckill/activities/%d", strings.TrimRight(adminBaseURL, "/"), activityID)
	statusCode, envelope, err := CallAPIEnvelope(DefaultHTTPClient(), "DELETE", requestURL, adminToken, nil)
	if err != nil {
		return err
	}
	if statusCode < 200 || statusCode >= 300 || strings.TrimSpace(envelope.Code) != "OK" {
		return fmt.Errorf("http=%d code=%s message=%s", statusCode, envelope.Code, envelope.Message)
	}
	return nil
}

func CreateTempActivityItem(adminBaseURL, adminToken string, activityID, productID int64) (int64, error) {
	requestURL := fmt.Sprintf("%s/api/v1/admin/seckill/activities/%d/items", strings.TrimRight(adminBaseURL, "/"), activityID)
	statusCode, envelope, err := CallAPIEnvelope(DefaultHTTPClient(), "POST", requestURL, adminToken, map[string]any{
		"product_id":            productID,
		"seckill_price_cent":    4900,
		"reserved_stock_total":  50000,
		"user_limit_mode":       0,
		"user_limit_window_sec": 0,
		"user_limit_qty":        0,
		"max_qty_per_order":     2,
		"status":                1,
	})
	if err != nil {
		return 0, err
	}
	if statusCode < 200 || statusCode >= 300 || strings.TrimSpace(envelope.Code) != "OK" {
		return 0, fmt.Errorf("create activity item failed: http=%d code=%s message=%s", statusCode, envelope.Code, envelope.Message)
	}

	var payload struct {
		Item struct {
			ItemID json.Number `json:"item_id"`
		} `json:"item"`
	}
	if err := json.Unmarshal(envelope.Data, &payload); err != nil {
		return 0, err
	}
	return ParseJSONNumberField(payload.Item.ItemID, "item_id")
}

func PublishTempActivity(adminBaseURL, adminToken string, activityID int64) error {
	requestURL := fmt.Sprintf("%s/api/v1/admin/seckill/activities/%d/publish", strings.TrimRight(adminBaseURL, "/"), activityID)
	statusCode, envelope, err := CallAPIEnvelope(DefaultHTTPClient(), "POST", requestURL, adminToken, map[string]any{})
	if err != nil {
		return err
	}
	if statusCode < 200 || statusCode >= 300 || strings.TrimSpace(envelope.Code) != "OK" {
		return fmt.Errorf("publish activity failed: http=%d code=%s message=%s", statusCode, envelope.Code, envelope.Message)
	}
	return nil
}

func CreateTempUsersAndTokens(userBaseURL, repoRoot string, userCount int, uniqueSuffix string) ([]int64, string, error) {
	if userCount <= 0 {
		return nil, "", nil
	}

	password := strings.TrimSpace(os.Getenv("FLASHSALE_PERF_USER_PASSWORD"))
	if password == "" {
		password = "abc12345"
	}

	tokens := make([]string, 0, userCount)
	userIDs := make([]int64, 0, userCount)
	for index := 0; index < userCount; index++ {
		phone := buildTempPhone(uniqueSuffix, index)
		nickname := fmt.Sprintf("perf_u_%d", index+1)
		sourceIP := BuildTempSourceIP(index)
		if err := registerTempUserWithRetry(userBaseURL, phone, password, nickname, sourceIP); err != nil {
			return nil, "", err
		}
		token, userID, err := loginTempUserWithRetry(userBaseURL, phone, password, sourceIP)
		if err != nil {
			return nil, "", err
		}
		tokens = append(tokens, token)
		userIDs = append(userIDs, userID)
	}

	tokenFile := filepath.Join(repoRoot, "log", "data", fmt.Sprintf("perf-temp-%s.tokens.txt", strings.TrimSpace(uniqueSuffix)))
	if err := os.MkdirAll(filepath.Dir(tokenFile), 0o755); err != nil {
		return nil, "", err
	}
	content := strings.Join(tokens, "\n")
	if content != "" {
		content += "\n"
	}
	if err := os.WriteFile(tokenFile, []byte(content), 0o600); err != nil {
		return nil, "", err
	}
	return userIDs, tokenFile, nil
}

func DeleteTempUser(adminBaseURL, adminToken string, userID int64) error {
	requestURL := fmt.Sprintf("%s/api/v1/admin/users/%d", strings.TrimRight(adminBaseURL, "/"), userID)
	statusCode, envelope, err := CallAPIEnvelope(DefaultHTTPClient(), "DELETE", requestURL, adminToken, nil)
	if err != nil {
		return err
	}
	if statusCode < 200 || statusCode >= 300 || strings.TrimSpace(envelope.Code) != "OK" {
		return fmt.Errorf("http=%d code=%s message=%s", statusCode, envelope.Code, envelope.Message)
	}
	return nil
}

func registerTempUserWithRetry(userBaseURL, phone, password, nickname, sourceIP string) error {
	maxAttempts := EnvIntWithFallback("FLASHSALE_PERF_USER_AUTH_MAX_ATTEMPTS", 8)
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		retryable, err := registerTempUser(userBaseURL, phone, password, nickname, sourceIP)
		if err == nil {
			return nil
		}
		lastErr = err
		if !retryable || attempt == maxAttempts {
			break
		}
		time.Sleep(TempUserRetryBackoff(attempt))
	}
	return lastErr
}

func registerTempUser(userBaseURL, phone, password, nickname, sourceIP string) (bool, error) {
	requestURL := strings.TrimRight(userBaseURL, "/") + "/api/v1/user/register"
	statusCode, envelope, err := CallAPIEnvelopeWithHeaders(DefaultHTTPClient(), "POST", requestURL, "", map[string]any{
		"phone":    phone,
		"password": password,
		"nickname": nickname,
	}, TempUserSourceHeaders(sourceIP))
	if err != nil {
		return true, err
	}
	if statusCode >= 200 && statusCode < 300 && strings.TrimSpace(envelope.Code) == "OK" {
		return false, nil
	}
	if statusCode == http.StatusConflict && strings.TrimSpace(envelope.Code) == "AUTH_PHONE_ALREADY_REGISTERED" {
		return false, nil
	}
	err = fmt.Errorf("register user failed: http=%d code=%s message=%s", statusCode, envelope.Code, envelope.Message)
	return ShouldRetryTempUserRequest(statusCode, envelope, nil), err
}

func loginTempUserWithRetry(userBaseURL, phone, password, sourceIP string) (string, int64, error) {
	maxAttempts := EnvIntWithFallback("FLASHSALE_PERF_USER_AUTH_MAX_ATTEMPTS", 8)
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		token, userID, retryable, err := loginTempUser(userBaseURL, phone, password, sourceIP)
		if err == nil {
			return token, userID, nil
		}
		lastErr = err
		if !retryable || attempt == maxAttempts {
			break
		}
		time.Sleep(TempUserRetryBackoff(attempt))
	}
	return "", 0, lastErr
}

func loginTempUser(userBaseURL, phone, password, sourceIP string) (string, int64, bool, error) {
	requestURL := strings.TrimRight(userBaseURL, "/") + "/api/v1/user/login"
	statusCode, envelope, err := CallAPIEnvelopeWithHeaders(DefaultHTTPClient(), "POST", requestURL, "", map[string]any{
		"phone":    phone,
		"password": password,
	}, TempUserSourceHeaders(sourceIP))
	if err != nil {
		return "", 0, true, err
	}
	if statusCode < 200 || statusCode >= 300 || strings.TrimSpace(envelope.Code) != "OK" {
		err = fmt.Errorf("login user failed: http=%d code=%s message=%s", statusCode, envelope.Code, envelope.Message)
		return "", 0, ShouldRetryTempUserRequest(statusCode, envelope, nil), err
	}

	var payload struct {
		AccessToken string      `json:"access_token"`
		UserID      json.Number `json:"user_id"`
	}
	if err := json.Unmarshal(envelope.Data, &payload); err != nil {
		return "", 0, false, err
	}
	userID, err := ParseJSONNumberField(payload.UserID, "user_id")
	if err != nil {
		return "", 0, false, err
	}
	return strings.TrimSpace(payload.AccessToken), userID, false, nil
}

func buildTempPhone(uniqueSuffix string, index int) string {
	base := strings.TrimSpace(uniqueSuffix) + strconv.Itoa(index+1) + strings.ReplaceAll(uuid.NewString(), "-", "")
	digits := make([]byte, 0, len(base))
	for i := 0; i < len(base); i++ {
		if base[i] >= '0' && base[i] <= '9' {
			digits = append(digits, base[i])
		}
	}
	if len(digits) < 9 {
		digits = append(digits, []byte("12345678901234567890")...)
	}
	core := digits[len(digits)-9:]
	return "13" + string(core)
}

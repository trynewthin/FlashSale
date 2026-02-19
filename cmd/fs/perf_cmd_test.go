package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestExtractPerfGlobalArgs(t *testing.T) {

	testCases := []struct {
		name        string
		args        []string
		wantEnvFile string
		wantArgs    []string
		wantErr     bool
	}{
		{
			name:        "env before subcommand with equals",
			args:        []string{"--env-file=configs/prod/server.env", "purchase-open", "-rate", "200"},
			wantEnvFile: "configs/prod/server.env",
			wantArgs:    []string{"purchase-open", "-rate", "200"},
		},
		{
			name:        "env before subcommand split",
			args:        []string{"--env-file", "configs/prod/server.env", "purchase-open", "-rate", "200"},
			wantEnvFile: "configs/prod/server.env",
			wantArgs:    []string{"purchase-open", "-rate", "200"},
		},
		{
			name:        "env after subcommand with equals",
			args:        []string{"purchase-open", "--env-file=configs/prod/server.env", "-rate", "200"},
			wantEnvFile: "configs/prod/server.env",
			wantArgs:    []string{"purchase-open", "-rate", "200"},
		},
		{
			name:        "env after subcommand split",
			args:        []string{"purchase-open", "--env-file", "configs/prod/server.env", "-rate", "200"},
			wantEnvFile: "configs/prod/server.env",
			wantArgs:    []string{"purchase-open", "-rate", "200"},
		},
		{
			name:        "single dash env style",
			args:        []string{"purchase-open", "-env-file=configs/prod/server.env", "-rate", "200"},
			wantEnvFile: "configs/prod/server.env",
			wantArgs:    []string{"purchase-open", "-rate", "200"},
		},
		{
			name:        "missing env value should error",
			args:        []string{"purchase-open", "--env-file"},
			wantEnvFile: "",
			wantArgs:    nil,
			wantErr:     true,
		},
		{
			name:        "fallback is used",
			args:        []string{"purchase-open", "-rate", "200"},
			wantEnvFile: "configs/deploy.env",
			wantArgs:    []string{"purchase-open", "-rate", "200"},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			global, gotArgs, err := extractPerfGlobalArgs(testCase.args)
			if testCase.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if global.EnvFile != testCase.wantEnvFile {
				t.Fatalf("env file mismatch: want=%s got=%s", testCase.wantEnvFile, global.EnvFile)
			}
			if len(gotArgs) != len(testCase.wantArgs) {
				t.Fatalf("args length mismatch: want=%d got=%d", len(testCase.wantArgs), len(gotArgs))
			}
			for index := range gotArgs {
				if gotArgs[index] != testCase.wantArgs[index] {
					t.Fatalf("arg mismatch at %d: want=%s got=%s", index, testCase.wantArgs[index], gotArgs[index])
				}
			}
		})
	}
}

func TestIsGatewayHealthy(t *testing.T) {

	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer okServer.Close()

	if !isGatewayHealthy(okServer.URL, 1000*time.Millisecond) {
		t.Fatalf("expected healthy gateway")
	}

	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer failServer.Close()

	if isGatewayHealthy(failServer.URL, 1000*time.Millisecond) {
		t.Fatalf("expected unhealthy gateway for 503")
	}
}

func TestExtractPerfControlArgs(t *testing.T) {

	options, args, err := extractPerfControlArgs([]string{
		"--auto-prepare",
		"-rate", "120",
		"--skip-preflight",
		"-timeout", "5s",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !options.AutoPrepare {
		t.Fatalf("expected auto-prepare=true")
	}
	if !options.SkipPreflight {
		t.Fatalf("expected skip-preflight=true")
	}
	want := []string{"-rate", "120", "-timeout", "5s"}
	if len(args) != len(want) {
		t.Fatalf("args length mismatch: want=%d got=%d", len(want), len(args))
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("arg mismatch at %d: want=%s got=%s", i, want[i], args[i])
		}
	}

	options, args, err = extractPerfControlArgs([]string{
		"--temp-users=5",
		"--keep-temp-data",
		"-requests", "1000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !options.KeepTempData {
		t.Fatalf("expected keep-temp-data=true")
	}
	if options.TempUsers != 5 {
		t.Fatalf("temp users mismatch: %d", options.TempUsers)
	}
	want = []string{"-requests", "1000"}
	if len(args) != len(want) {
		t.Fatalf("args length mismatch: want=%d got=%d", len(want), len(args))
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("arg mismatch at %d: want=%s got=%s", i, want[i], args[i])
		}
	}

	options, args, err = extractPerfControlArgs([]string{"-requests", "1000"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if options.TempUsers != 0 {
		t.Fatalf("default temp users should be auto(0), got=%d", options.TempUsers)
	}
	if options.KeepTempData {
		t.Fatalf("keep-temp-data should default false")
	}
	if len(args) != 2 || args[0] != "-requests" || args[1] != "1000" {
		t.Fatalf("args passthrough mismatch: %v", args)
	}
}

func TestMergePerfArgsWithTempData(t *testing.T) {
	temp := &perfTempData{
		ActivityID: 100,
		ItemID:     200,
		TokenFile:  "log/data/temp.tokens.txt",
	}
	got := mergePerfArgsWithTempData([]string{
		"-activity-id", "1",
		"--item-id=2",
		"-token", "abc",
		"-token-file", "old.txt",
		"-rate", "120",
	}, perfScenarioPurchaseOpen, temp)
	gotJoined := strings.Join(got, " ")
	if strings.Contains(gotJoined, "old.txt") || strings.Contains(gotJoined, " -token abc") {
		t.Fatalf("old token args should be removed: %s", gotJoined)
	}
	if !strings.Contains(gotJoined, "-activity-id 100") || !strings.Contains(gotJoined, "-item-id 200") || !strings.Contains(gotJoined, "-token-file log/data/temp.tokens.txt") {
		t.Fatalf("temp args not injected: %s", gotJoined)
	}
}

func TestBuildPerfReadinessConfig(t *testing.T) {

	t.Setenv("FLASHSALE_BASE_URL", "http://127.0.0.1:18000")
	t.Setenv("FLASHSALE_ACTIVITY_ID", "100")
	t.Setenv("FLASHSALE_ACTIVITY_ITEM_ID", "200")
	t.Setenv("FLASHSALE_PRESSURE_QUANTITY", "1")
	t.Setenv("FLASHSALE_USER_TOKEN", "token-from-env")
	t.Setenv("FLASHSALE_TOKENS_FILE", "")
	t.Setenv("FLASHSALE_HTTP_TIMEOUT", "3s")

	cfg, err := buildPerfReadinessConfig(perfScenarioPurchaseOpen, []string{
		"--base-url=http://127.0.0.1:8082",
		"--activity-id", "900",
		"--item-id=901",
		"--quantity", "2",
		"--token", "token-from-arg",
		"--timeout", "7s",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BaseURL != "http://127.0.0.1:8082" {
		t.Fatalf("base-url mismatch: %s", cfg.BaseURL)
	}
	if cfg.ActivityID != 900 || cfg.ActivityItemID != 901 {
		t.Fatalf("id mismatch: activity=%d item=%d", cfg.ActivityID, cfg.ActivityItemID)
	}
	if cfg.Quantity != 2 {
		t.Fatalf("quantity mismatch: %d", cfg.Quantity)
	}
	if cfg.Token != "token-from-arg" {
		t.Fatalf("token mismatch: %s", cfg.Token)
	}
	if cfg.Timeout != 7*time.Second {
		t.Fatalf("timeout mismatch: %s", cfg.Timeout)
	}
}

func TestRunPerfPreflightOnce(t *testing.T) {

	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/healthz":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		case r.URL.Path == "/api/v1/seckill/activities/100":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":"OK","message":"ok","data":{"activity":{"activity_id":100,"start_at_unix":1000,"end_at_unix":4102444800,"status":1,"items":[{"item_id":200,"in_stock":true,"max_qty_per_order":2}]}}}`))
		case strings.HasPrefix(r.URL.Path, "/api/v1/orders"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":"OK","message":"ok","data":{"list":[],"total":0,"page":1,"page_size":1}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer okServer.Close()

	t.Setenv("FLASHSALE_BASE_URL", okServer.URL)
	t.Setenv("FLASHSALE_ACTIVITY_ID", "100")
	t.Setenv("FLASHSALE_ACTIVITY_ITEM_ID", "200")
	t.Setenv("FLASHSALE_PRESSURE_QUANTITY", "1")
	t.Setenv("FLASHSALE_USER_TOKEN", "mock-token")
	t.Setenv("FLASHSALE_HTTP_TIMEOUT", "2s")
	if err := runPerfPreflightOnce(".", perfScenarioPurchaseOpen, nil); err != nil {
		t.Fatalf("expected preflight success, got error: %v", err)
	}
}

func TestRunPerfPreflightOnceFailOnActivityState(t *testing.T) {

	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/healthz":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		case r.URL.Path == "/api/v1/seckill/activities/100":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":"SECKILL_ACTIVITY_NOT_PUBLISHED","message":"活动未发布","data":{}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer failServer.Close()

	t.Setenv("FLASHSALE_BASE_URL", failServer.URL)
	t.Setenv("FLASHSALE_ACTIVITY_ID", "100")
	t.Setenv("FLASHSALE_ACTIVITY_ITEM_ID", "200")
	t.Setenv("FLASHSALE_PRESSURE_QUANTITY", "1")
	t.Setenv("FLASHSALE_USER_TOKEN", "mock-token")
	t.Setenv("FLASHSALE_HTTP_TIMEOUT", "2s")
	err := runPerfPreflightOnce(".", perfScenarioPurchaseOpen, nil)
	if err == nil {
		t.Fatalf("expected preflight error")
	}
	if !strings.Contains(err.Error(), "SECKILL_ACTIVITY_NOT_PUBLISHED") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeriveSeedGatewayBaseURLs(t *testing.T) {

	t.Setenv("FLASHSALE_USER_BASE_URL", "")
	t.Setenv("FLASHSALE_ADMIN_BASE_URL", "")
	userURL, adminURL := deriveSeedGatewayBaseURLs("http://127.0.0.1:8082")
	if userURL != "http://127.0.0.1:8082" {
		t.Fatalf("unexpected userURL: %s", userURL)
	}
	if adminURL != "http://127.0.0.1:8083" {
		t.Fatalf("unexpected adminURL: %s", adminURL)
	}
}

func TestFindPerfFlagValue(t *testing.T) {

	value, found, err := findPerfFlagValue([]string{"-rate", "120", "--timeout=5s"}, "timeout")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || value != "5s" {
		t.Fatalf("unexpected value: found=%v value=%s", found, value)
	}

	_, _, err = findPerfFlagValue([]string{"--timeout"}, "timeout")
	if err == nil {
		t.Fatalf("expected missing-value error")
	}
	if !strings.Contains(err.Error(), "missing value") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolvePerfTokenFromFile(t *testing.T) {

	dir := t.TempDir()
	tokenFile := dir + "/tokens.txt"
	content := "# comment\n\nabc-token\n"
	if err := os.WriteFile(tokenFile, []byte(content), 0o600); err != nil {
		t.Fatalf("write token file failed: %v", err)
	}
	token, source, err := resolvePerfToken("", tokenFile, ".")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "abc-token" {
		t.Fatalf("token mismatch: %s", token)
	}
	if !strings.Contains(source, "tokens.txt") {
		t.Fatalf("source mismatch: %s", source)
	}
}

func TestValidateActivityWindowForPreflight(t *testing.T) {

	activity := activityPublicForPreflight{
		ActivityID:  1,
		StartAtUnix: time.Now().Add(-time.Minute).Unix(),
		EndAtUnix:   time.Now().Add(time.Minute).Unix(),
		Status:      1,
	}
	if err := validateActivityWindowForPreflight(activity); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expired := activity
	expired.EndAtUnix = time.Now().Add(-time.Second).Unix()
	err := validateActivityWindowForPreflight(expired)
	if err == nil || !strings.Contains(err.Error(), "已结束") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchActivityPublicForPreflightDecodeError(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/seckill/activities/1":
			_, _ = fmt.Fprintln(w, `{"code":"OK","message":"ok","data":"bad"}`)
		case "/healthz":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &http.Client{Timeout: time.Second}
	_, err := fetchActivityPublicForPreflight(client, server.URL, 1)
	if err == nil {
		t.Fatalf("expected decode error")
	}
}

func TestResolveTempUserCount(t *testing.T) {
	t.Setenv("FLASHSALE_PRESSURE_CONCURRENCY", "120")
	count := resolveTempUserCount(perfScenarioPurchaseOpen, 0, []string{"--concurrency", "80"})
	if count != 80 {
		t.Fatalf("expected 80, got=%d", count)
	}
	count = resolveTempUserCount(perfScenarioPurchaseOpen, 300, nil)
	if count != 300 {
		t.Fatalf("expected configured 300, got=%d", count)
	}
	count = resolveTempUserCount(perfScenarioIdempotency, 99, []string{"--concurrency", "200"})
	if count != 1 {
		t.Fatalf("idempotency should force 1, got=%d", count)
	}
	count = resolveTempUserCount(perfScenarioPurchaseStress, 0, []string{"--concurrency", "2000"})
	if count != 500 {
		t.Fatalf("should cap to 500, got=%d", count)
	}
}

func TestBuildPerfTempSourceIP(t *testing.T) {
	first := buildPerfTempSourceIP(0)
	second := buildPerfTempSourceIP(1)
	rolled := buildPerfTempSourceIP(250)
	if first == second {
		t.Fatalf("source ip should vary by index: first=%s second=%s", first, second)
	}
	if !strings.HasPrefix(first, "10.77.") || !strings.HasPrefix(rolled, "10.77.") {
		t.Fatalf("unexpected source ip prefix: first=%s rolled=%s", first, rolled)
	}
}

func TestShouldRetryPerfTempUserRequest(t *testing.T) {
	if !shouldRetryPerfTempUserRequest(http.StatusTooManyRequests, perfAPIEnvelope{}, nil) {
		t.Fatalf("429 should be retryable")
	}
	if !shouldRetryPerfTempUserRequest(http.StatusInternalServerError, perfAPIEnvelope{}, nil) {
		t.Fatalf("5xx should be retryable")
	}
	if !shouldRetryPerfTempUserRequest(http.StatusBadRequest, perfAPIEnvelope{
		Code:    "SYS_BAD_REQUEST",
		Message: "请求过于频繁，请稍后重试",
	}, nil) {
		t.Fatalf("rate-limit bad request should be retryable")
	}
	if shouldRetryPerfTempUserRequest(http.StatusConflict, perfAPIEnvelope{
		Code:    "AUTH_PHONE_ALREADY_REGISTERED",
		Message: "手机号已注册",
	}, nil) {
		t.Fatalf("conflict should not be retryable")
	}
}

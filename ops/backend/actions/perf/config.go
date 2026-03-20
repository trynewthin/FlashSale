package perf

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	ScenarioPurchaseStress = "purchase-stress"
	ScenarioIdempotency    = "idempotency"
	ScenarioTrackStress    = "track-stress"
	ScenarioPurchaseOpen   = "purchase-open"
	ScenarioTrackOpen      = "track-open"
)

type ControlOptions struct {
	SkipPreflight bool
	AutoPrepare   bool
	KeepTempData  bool
	TempUsers     int
}

type GlobalArgs struct {
	EnvFile      string
	AdminBaseURL string
	UserBaseURL  string
}

func ExtractPerfControlArgs(args []string) (ControlOptions, []string, error) {
	options := ControlOptions{TempUsers: 0}
	cleanArgs := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		rawToken := args[index]
		token := strings.TrimSpace(rawToken)
		switch {
		case token == "--skip-preflight":
			options.SkipPreflight = true
		case token == "--auto-prepare":
			options.AutoPrepare = true
		case token == "--keep-temp-data":
			options.KeepTempData = true
		case token == "--temp-users":
			if index+1 >= len(args) {
				return options, nil, fmt.Errorf("missing value for --temp-users")
			}
			value, err := ParsePositiveInt(strings.TrimSpace(args[index+1]), "--temp-users")
			if err != nil {
				return options, nil, err
			}
			options.TempUsers = value
			index++
		case strings.HasPrefix(token, "--temp-users="):
			value, err := ParsePositiveInt(strings.TrimSpace(strings.TrimPrefix(token, "--temp-users=")), "--temp-users")
			if err != nil {
				return options, nil, err
			}
			options.TempUsers = value
		default:
			cleanArgs = append(cleanArgs, rawToken)
		}
	}
	return options, cleanArgs, nil
}

func ParsePositiveInt(raw, name string) (int, error) {
	if raw == "" {
		return 0, fmt.Errorf("%s cannot be empty", name)
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s invalid: %w", name, err)
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s must be greater than 0", name)
	}
	return value, nil
}

func ExtractPerfGlobalArgs(args []string) (GlobalArgs, []string, error) {
	global := GlobalArgs{EnvFile: "configs/deploy.env"}
	cleanArgs := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		token := strings.TrimSpace(args[index])
		switch {
		case token == "--env-file" || token == "-env-file":
			if index+1 >= len(args) {
				return global, nil, fmt.Errorf("missing value for %s", token)
			}
			global.EnvFile = strings.TrimSpace(args[index+1])
			index++
		case strings.HasPrefix(token, "--env-file="):
			global.EnvFile = strings.TrimSpace(strings.TrimPrefix(token, "--env-file="))
		case strings.HasPrefix(token, "-env-file="):
			global.EnvFile = strings.TrimSpace(strings.TrimPrefix(token, "-env-file="))
		case token == "--admin-base-url":
			if index+1 >= len(args) {
				return global, nil, fmt.Errorf("missing value for --admin-base-url")
			}
			global.AdminBaseURL = strings.TrimSpace(args[index+1])
			index++
		case strings.HasPrefix(token, "--admin-base-url="):
			global.AdminBaseURL = strings.TrimSpace(strings.TrimPrefix(token, "--admin-base-url="))
		case token == "--user-base-url":
			if index+1 >= len(args) {
				return global, nil, fmt.Errorf("missing value for --user-base-url")
			}
			global.UserBaseURL = strings.TrimSpace(args[index+1])
			index++
		case strings.HasPrefix(token, "--user-base-url="):
			global.UserBaseURL = strings.TrimSpace(strings.TrimPrefix(token, "--user-base-url="))
		default:
			cleanArgs = append(cleanArgs, args[index])
		}
	}
	return global, cleanArgs, nil
}

func EnsureBaseURL() {
	if strings.TrimSpace(os.Getenv("FLASHSALE_BASE_URL")) != "" {
		return
	}
	if userBaseURL := strings.TrimSpace(os.Getenv("FLASHSALE_USER_BASE_URL")); userBaseURL != "" {
		_ = os.Setenv("FLASHSALE_BASE_URL", userBaseURL)
		return
	}

	candidates := []string{
		"http://127.0.0.1:18000",
		"http://127.0.0.1:8082",
	}
	for _, baseURL := range candidates {
		if IsGatewayHealthy(baseURL, 1200*time.Millisecond) {
			_ = os.Setenv("FLASHSALE_BASE_URL", baseURL)
			return
		}
	}
}

func IsGatewayHealthy(baseURL string, timeout time.Duration) bool {
	url := strings.TrimRight(baseURL, "/") + "/healthz"
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func ApplyDefaultsFromRunlogs(repoRoot string) {
	candidates := []struct {
		key  string
		path string
	}{
		{"FLASHSALE_ACTIVITY_ID", filepath.Join(repoRoot, "log", "data", "perf.activity_id.txt")},
		{"FLASHSALE_ACTIVITY_ITEM_ID", filepath.Join(repoRoot, "log", "data", "perf.item_id.txt")},
		{"FLASHSALE_USER_TOKEN", filepath.Join(repoRoot, "log", "data", "user.token.txt")},
	}
	for _, candidate := range candidates {
		SetEnvFromFileIfEmpty(candidate.key, candidate.path)
	}
	if strings.TrimSpace(os.Getenv("FLASHSALE_TOKENS_FILE")) == "" {
		tokenPath := filepath.Join(repoRoot, "log", "data", "user.token.txt")
		if _, err := os.Stat(tokenPath); err == nil {
			_ = os.Setenv("FLASHSALE_TOKENS_FILE", tokenPath)
		}
	}
}

func SetEnvFromFileIfEmpty(key, path string) {
	if strings.TrimSpace(os.Getenv(key)) != "" {
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	value := strings.TrimSpace(string(raw))
	if value == "" {
		return
	}
	_ = os.Setenv(key, value)
}

func FindSeckillloadBinary() string {
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "seckillload")
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return candidate
		}
	}
	if info, err := os.Stat("/app/bin/seckillload"); err == nil && !info.IsDir() {
		return "/app/bin/seckillload"
	}
	if path, err := exec.LookPath("seckillload"); err == nil {
		return path
	}
	return ""
}

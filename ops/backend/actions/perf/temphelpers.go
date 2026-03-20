package perf

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const maxAutoTempUsers = 500

func MergeArgsWithTempData(perfArgs []string, scenario string, activityID, itemID int64, tokenFile string) []string {
	out := StripFlags(perfArgs, "activity-id", "item-id", "token", "token-file")
	out = append(out, "-activity-id", strconv.FormatInt(activityID, 10))
	out = append(out, "-item-id", strconv.FormatInt(itemID, 10))
	if RequiresPurchaseToken(scenario) && strings.TrimSpace(tokenFile) != "" {
		out = append(out, "-token-file", tokenFile)
	}
	return out
}

func StripFlags(args []string, names ...string) []string {
	if len(names) == 0 {
		return append([]string{}, args...)
	}
	nameSet := make(map[string]struct{}, len(names)*2)
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		nameSet["-"+name] = struct{}{}
		nameSet["--"+name] = struct{}{}
	}

	out := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		token := strings.TrimSpace(args[index])
		if _, ok := nameSet[token]; ok {
			if index+1 < len(args) && !strings.HasPrefix(strings.TrimSpace(args[index+1]), "-") {
				index++
			}
			continue
		}
		skip := false
		for key := range nameSet {
			if strings.HasPrefix(token, key+"=") {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		out = append(out, args[index])
	}
	return out
}

func BuildTempSourceIP(index int) string {
	if index < 0 {
		index = -index
	}
	third := (index / 250 % 250) + 1
	fourth := (index % 250) + 1
	return fmt.Sprintf("10.77.%d.%d", third, fourth)
}

func TempUserSourceHeaders(sourceIP string) map[string]string {
	ip := strings.TrimSpace(sourceIP)
	if ip == "" {
		return nil
	}
	return map[string]string{
		"X-Forwarded-For": ip,
		"X-Real-IP":       ip,
	}
}

func TempUserRetryBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	return time.Duration(attempt) * 120 * time.Millisecond
}

func ShouldRetryTempUserRequest(statusCode int, envelope APIEnvelope, callErr error) bool {
	if callErr != nil {
		return true
	}
	if statusCode == http.StatusTooManyRequests {
		return true
	}
	if statusCode >= http.StatusInternalServerError {
		return true
	}
	code := strings.TrimSpace(envelope.Code)
	message := strings.TrimSpace(envelope.Message)
	if code == "SYS_BAD_REQUEST" && (strings.Contains(message, "retry") || strings.Contains(message, "rate") || strings.Contains(message, "frequent")) {
		return true
	}
	return false
}

func DefaultHTTPClient() *http.Client {
	timeout := EnvDurationWithFallback("FLASHSALE_HTTP_TIMEOUT", DefaultTimeout)
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &http.Client{Timeout: timeout}
}

func ResolveTempUserCount(scenario string, configured int, perfArgs []string) int {
	if scenario == ScenarioIdempotency {
		return 1
	}
	if configured > 0 {
		return configured
	}

	concurrency := EnvIntWithFallback("FLASHSALE_PRESSURE_CONCURRENCY", 100)
	if value, found, err := FindFlagValue(perfArgs, "concurrency"); err == nil && found {
		if parsed, parseErr := strconv.Atoi(strings.TrimSpace(value)); parseErr == nil && parsed > 0 {
			concurrency = parsed
		}
	}
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > maxAutoTempUsers {
		return maxAutoTempUsers
	}
	return concurrency
}

func EnvIntWithFallback(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

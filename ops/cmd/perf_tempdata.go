package main

import (
	"net/http"
	"time"

	perfaction "flashsale/ops/backend/actions/perf"
)

type perfTempData = perfaction.TempData

func setupPerfTempData(repoRoot, scenario string, perfArgs []string, options perfControlOptions) (*perfTempData, error) {
	return perfaction.SetupTempData(repoRoot, scenario, perfArgs, options)
}

func cleanupPerfTempData(temp *perfTempData) error {
	return perfaction.CleanupTempData(temp)
}

func mergePerfArgsWithTempData(perfArgs []string, scenario string, temp *perfTempData) []string {
	if temp == nil {
		return perfArgs
	}
	return perfaction.MergeArgsWithTempData(perfArgs, scenario, temp.ActivityID, temp.ItemID, temp.TokenFile)
}

func stripPerfFlags(args []string, names ...string) []string {
	return perfaction.StripFlags(args, names...)
}

func buildPerfTempSourceIP(index int) string {
	return perfaction.BuildTempSourceIP(index)
}

func perfTempUserSourceHeaders(sourceIP string) map[string]string {
	return perfaction.TempUserSourceHeaders(sourceIP)
}

func perfTempUserRetryBackoff(attempt int) time.Duration {
	return perfaction.TempUserRetryBackoff(attempt)
}

func shouldRetryPerfTempUserRequest(statusCode int, envelope perfAPIEnvelope, callErr error) bool {
	return perfaction.ShouldRetryTempUserRequest(statusCode, envelope, callErr)
}

func defaultPerfHTTPClient() *http.Client {
	return perfaction.DefaultHTTPClient()
}

func resolveTempUserCount(scenario string, configured int, perfArgs []string) int {
	return perfaction.ResolveTempUserCount(scenario, configured, perfArgs)
}

func envIntFromEnv(key string, fallback int) int {
	return perfaction.EnvIntWithFallback(key, fallback)
}

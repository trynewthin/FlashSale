package main

import (
	"encoding/json"
	"net/http"
	"time"

	perfaction "flashsale/ops/backend/actions/perf"
)

const perfDefaultTimeout = perfaction.DefaultTimeout

type perfReadinessConfig = perfaction.ReadinessConfig
type perfAPIEnvelope = perfaction.APIEnvelope
type activityPublicForPreflight = perfaction.ActivityPublicForPreflight
type activityItemPublicForPreflight = perfaction.ActivityItemPublicForPreflight

func runPerfPrepare(args []string, envFile string) error {
	return perfaction.RunPrepare(args, envFile, perfaction.PrepareDeps{
		RunSmoke: func(envFile string) error {
			return runEnvSmoke([]string{"--env-file=" + envFile})
		},
		RunSeedOverwrite: func(envFile, userBaseURL, adminBaseURL string) error {
			return runData([]string{
				"seed-overwrite",
				"--force",
				"--env-file=" + envFile,
				"--user-base-url=" + userBaseURL,
				"--admin-base-url=" + adminBaseURL,
			})
		},
	})
}

func runPerfPreflight(repoRoot, envFile, scenario string, perfArgs []string, autoPrepare bool) error {
	return perfaction.RunPreflight(repoRoot, envFile, scenario, perfArgs, autoPrepare, runPerfPrepare)
}

func runPerfPreflightOnce(repoRoot, scenario string, perfArgs []string) error {
	return perfaction.RunPreflightOnce(repoRoot, scenario, perfArgs)
}

func buildPerfReadinessConfig(scenario string, perfArgs []string) (perfReadinessConfig, error) {
	return perfaction.BuildReadinessConfig(scenario, perfArgs)
}

func findPerfFlagValue(args []string, flagName string) (string, bool, error) {
	return perfaction.FindFlagValue(args, flagName)
}

func fetchActivityPublicForPreflight(client *http.Client, baseURL string, activityID int64) (activityPublicForPreflight, error) {
	return perfaction.FetchActivityPublicForPreflight(client, baseURL, activityID)
}

func callPerfAPIEnvelope(client *http.Client, method, requestURL, token string, body any) (int, perfAPIEnvelope, error) {
	return perfaction.CallAPIEnvelope(client, method, requestURL, token, body)
}

func callPerfAPIEnvelopeWithHeaders(client *http.Client, method, requestURL, token string, body any, extraHeaders map[string]string) (int, perfAPIEnvelope, error) {
	return perfaction.CallAPIEnvelopeWithHeaders(client, method, requestURL, token, body, extraHeaders)
}

func validateActivityWindowForPreflight(activity activityPublicForPreflight) error {
	return perfaction.ValidateActivityWindowForPreflight(activity)
}

func findActivityItemForPreflight(items []activityItemPublicForPreflight, itemID int64) (activityItemPublicForPreflight, error) {
	return perfaction.FindActivityItemForPreflight(items, itemID)
}

func validatePurchaseTokenForPreflight(client *http.Client, baseURL, token string) error {
	return perfaction.ValidatePurchaseTokenForPreflight(client, baseURL, token)
}

func resolvePerfToken(token, tokenFile, repoRoot string) (string, string, error) {
	return perfaction.ResolveToken(token, tokenFile, repoRoot)
}

func deriveSeedGatewayBaseURLs(baseURL string) (string, string) {
	return perfaction.DeriveSeedGatewayBaseURLs(baseURL)
}

func parseJSONNumberField(value json.Number, field string) (int64, error) {
	return perfaction.ParseJSONNumberField(value, field)
}

func requiresPurchaseToken(scenario string) bool {
	return perfaction.RequiresPurchaseToken(scenario)
}

func envInt64WithFallback(key string, fallback int64) int64 {
	return perfaction.EnvInt64WithFallback(key, fallback)
}

func envDurationWithFallback(key string, fallback time.Duration) time.Duration {
	return perfaction.EnvDurationWithFallback(key, fallback)
}

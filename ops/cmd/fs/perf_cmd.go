package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	perfaction "flashsale/ops/backend/actions/perf"
	"flashsale/ops/backend/shared/devenv"
	"flashsale/ops/backend/shared/platform"
)

const (
	perfScenarioPurchaseStress = perfaction.ScenarioPurchaseStress
	perfScenarioIdempotency    = perfaction.ScenarioIdempotency
	perfScenarioTrackStress    = perfaction.ScenarioTrackStress
	perfScenarioPurchaseOpen   = perfaction.ScenarioPurchaseOpen
	perfScenarioTrackOpen      = perfaction.ScenarioTrackOpen
)

type perfControlOptions = perfaction.ControlOptions
type perfGlobalArgs = perfaction.GlobalArgs

func runPerf(args []string) error {
	if len(args) == 0 {
		printPerfUsage()
		return nil
	}

	global, rest, err := extractPerfGlobalArgs(args)
	if err != nil {
		return err
	}
	envFile := global.EnvFile
	if global.AdminBaseURL != "" {
		_ = os.Setenv("FLASHSALE_ADMIN_BASE_URL", global.AdminBaseURL)
	}
	if global.UserBaseURL != "" {
		_ = os.Setenv("FLASHSALE_USER_BASE_URL", global.UserBaseURL)
	}

	if len(rest) == 0 {
		printPerfUsage()
		return nil
	}
	if rest[0] == "prepare" {
		return runPerfPrepare(rest[1:], envFile)
	}

	scenario := ""
	switch rest[0] {
	case perfScenarioPurchaseStress:
		scenario = perfScenarioPurchaseStress
	case perfScenarioIdempotency:
		scenario = perfScenarioIdempotency
	case perfScenarioTrackStress:
		scenario = perfScenarioTrackStress
	case perfScenarioPurchaseOpen:
		scenario = perfScenarioPurchaseOpen
	case perfScenarioTrackOpen:
		scenario = perfScenarioTrackOpen
	default:
		printPerfUsage()
		return fmt.Errorf("unknown perf subcommand: %s", rest[0])
	}

	controlOptions, perfArgs, err := extractPerfControlArgs(rest[1:])
	if err != nil {
		return err
	}

	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	_ = devenv.Load(devenv.ResolvePath(repoRoot, envFile))
	ensurePerfBaseURL()
	applyPerfDefaultsFromRunlogs(repoRoot)

	tempData, err := setupPerfTempData(repoRoot, scenario, perfArgs, controlOptions)
	if err != nil {
		return err
	}
	if !controlOptions.KeepTempData {
		defer func() {
			if cleanupErr := cleanupPerfTempData(tempData); cleanupErr != nil {
				fmt.Fprintf(os.Stderr, "WARN: temp data cleanup failed: %v\n", cleanupErr)
			}
		}()
	}

	perfArgs = mergePerfArgsWithTempData(perfArgs, scenario, tempData)
	if !controlOptions.SkipPreflight {
		if err := runPerfPreflight(repoRoot, envFile, scenario, perfArgs, controlOptions.AutoPrepare); err != nil {
			return err
		}
	}

	seckillloadBin := findSeckillloadBinary()
	if seckillloadBin != "" {
		cmdArgs := []string{"-scenario", scenario}
		cmdArgs = append(cmdArgs, perfArgs...)
		return platform.Run(context.Background(), repoRoot, seckillloadBin, cmdArgs...)
	}
	cmdArgs := []string{"run", "./ops/executor/seckillload", "-scenario", scenario}
	cmdArgs = append(cmdArgs, perfArgs...)
	return platform.Run(context.Background(), repoRoot, "go", cmdArgs...)
}

func extractPerfControlArgs(args []string) (perfControlOptions, []string, error) {
	return perfaction.ExtractPerfControlArgs(args)
}

func parsePositiveInt(raw, name string) (int, error) {
	return perfaction.ParsePositiveInt(raw, name)
}

func extractPerfGlobalArgs(args []string) (perfGlobalArgs, []string, error) {
	return perfaction.ExtractPerfGlobalArgs(args)
}

func ensurePerfBaseURL() {
	perfaction.EnsureBaseURL()
}

func isGatewayHealthy(baseURL string, timeout time.Duration) bool {
	return perfaction.IsGatewayHealthy(baseURL, timeout)
}

func applyPerfDefaultsFromRunlogs(repoRoot string) {
	perfaction.ApplyDefaultsFromRunlogs(repoRoot)
}

func setEnvFromFileIfEmpty(key, path string) {
	perfaction.SetEnvFromFileIfEmpty(key, path)
}

func printPerfUsage() {
	fmt.Print(`fs perf usage:
  fs perf [--env-file configs/deploy.env] prepare [--skip-smoke] [--skip-seed] [--base-url http://127.0.0.1:18000]
  fs perf [--env-file configs/deploy.env] purchase-stress [seckillload flags...]
  fs perf [--env-file configs/deploy.env] idempotency [seckillload flags...]
  fs perf [--env-file configs/deploy.env] track-stress [seckillload flags...]
  fs perf [--env-file configs/deploy.env] purchase-open [seckillload flags...]
  fs perf [--env-file configs/deploy.env] track-open [seckillload flags...]

Notes:
  - temp data is created automatically for each perf run.
  - use --keep-temp-data to keep temp data after the run.
  - use --temp-users N to control temp user count for purchase scenarios.
  - preflight runs by default; use --skip-preflight to disable it.
  - use --auto-prepare to run prepare automatically after a preflight failure.
  - prepare runs env smoke -> data seed-overwrite -> readiness check.
` + "\n")
}

func findSeckillloadBinary() string {
	return perfaction.FindSeckillloadBinary()
}


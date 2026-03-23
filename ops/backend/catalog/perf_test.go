package catalog

import "testing"

func TestBuildPerfJobArgsAllowsExpandedPurchaseScale(t *testing.T) {
	args, err := BuildPerfJobArgs("perf.purchase_stress", map[string]string{
		"concurrency": "100000",
		"temp-users":  "1000000",
		"requests":    "10000000",
		"timeout":     "7s",
		"output":      "json",
	}, "")
	if err != nil {
		t.Fatalf("BuildPerfJobArgs returned error: %v", err)
	}
	if len(args) == 0 {
		t.Fatal("expected args to be built")
	}
}

func TestBuildPerfJobArgsAllowsExpandedTrackOpenScale(t *testing.T) {
	args, err := BuildPerfJobArgs("perf.track_open", map[string]string{
		"rate":          "100000",
		"open-duration": "30s",
		"concurrency":   "100000",
		"timeout":       "4s",
		"event-type":    "pv",
		"output":        "json",
	}, "")
	if err != nil {
		t.Fatalf("BuildPerfJobArgs returned error: %v", err)
	}
	if len(args) == 0 {
		t.Fatal("expected args to be built")
	}
}

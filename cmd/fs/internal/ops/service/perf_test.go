package service

import (
	"strings"
	"testing"
)

func TestBuildPerfJobArgs(t *testing.T) {
	args, err := BuildPerfJobArgs("perf.purchase_open", map[string]string{
		"rate":          "160",
		"open-duration": "45s",
		"concurrency":   "400",
		"timeout":       "7s",
		"output":        "json",
		"temp-users":    "120",
	}, "--keep-temp-data --max-network-errors 10")
	if err != nil {
		t.Fatalf("build args failed: %v", err)
	}
	joined := strings.Join(args, " ")
	mustContain := []string{
		"--rate 160",
		"--open-duration 45s",
		"--concurrency 400",
		"--timeout 7s",
		"--output json",
		"--temp-users 120",
		"--keep-temp-data",
		"--max-network-errors 10",
	}
	for _, expected := range mustContain {
		if !strings.Contains(joined, expected) {
			t.Fatalf("args missing %q, got: %s", expected, joined)
		}
	}
}

func TestBuildPerfJobArgsRejectUnknownField(t *testing.T) {
	_, err := BuildPerfJobArgs("perf.track_open", map[string]string{
		"rate":        "1000",
		"unexpected":  "x",
		"concurrency": "300",
		"timeout":     "4s",
		"event-type":  "pv",
		"output":      "json",
	}, "")
	if err == nil || !strings.Contains(err.Error(), "不支持的参数字段") {
		t.Fatalf("expected unknown-field error, got: %v", err)
	}
}

func TestBuildPerfJobArgsValidateFieldValue(t *testing.T) {
	_, err := BuildPerfJobArgs("perf.idempotency", map[string]string{
		"concurrency":        "50",
		"requests":           "200",
		"expect-max-success": "20",
		"output":             "json",
	}, "")
	if err == nil || !strings.Contains(err.Error(), "不能大于") {
		t.Fatalf("expected range error, got: %v", err)
	}
}

func TestBuildPerfJobArgsRequireField(t *testing.T) {
	_, err := BuildPerfJobArgs("perf.track_stress", map[string]string{
		"concurrency": "300",
		"requests":    "6000",
		"timeout":     "4s",
		"output":      "json",
	}, "")
	if err == nil || !strings.Contains(err.Error(), "event-type 不能为空") {
		t.Fatalf("expected required-field error, got: %v", err)
	}
}

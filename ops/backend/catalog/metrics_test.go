package catalog

import "testing"

func TestResolveMetricQueryOverviewKeepsExistingWindow(t *testing.T) {
	query, err := resolveMetricQuery("rpc_p99_latency", MetricProfileOverview, "", "")
	if err != nil {
		t.Fatalf("resolveMetricQuery failed: %v", err)
	}
	want := `histogram_quantile(0.99, sum by (le) (rate(rpc_server_requests_duration_ms_bucket[5m])))`
	if query != want {
		t.Fatalf("overview query = %q, want %q", query, want)
	}
}

func TestResolveMetricQueryBurstUsesShortWindow(t *testing.T) {
	query, err := resolveMetricQuery("rpc_request_rate", MetricProfileBurst, "", "")
	if err != nil {
		t.Fatalf("resolveMetricQuery failed: %v", err)
	}
	want := `sum(rate(rpc_server_requests_duration_ms_count[30s]))`
	if query != want {
		t.Fatalf("burst query = %q, want %q", query, want)
	}
}

func TestResolveMetricQueryReplayUsesAdaptiveWindow(t *testing.T) {
	cases := []struct {
		name    string
		start   string
		end     string
		wantHas string
	}{
		{
			name:    "short window",
			start:   "2026-03-21T10:00:00Z",
			end:     "2026-03-21T10:10:00Z",
			wantHas: "[30s]",
		},
		{
			name:    "medium window",
			start:   "2026-03-21T10:00:00Z",
			end:     "2026-03-21T11:00:00Z",
			wantHas: "[1m]",
		},
		{
			name:    "day window",
			start:   "2026-03-21T00:00:00Z",
			end:     "2026-03-21T12:00:00Z",
			wantHas: "[5m]",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			query, err := resolveMetricQuery("rpc_error_rate", MetricProfileReplay, tc.start, tc.end)
			if err != nil {
				t.Fatalf("resolveMetricQuery failed: %v", err)
			}
			if !contains(query, tc.wantHas) {
				t.Fatalf("replay query = %q, want substring %q", query, tc.wantHas)
			}
		})
	}
}

func TestResolveMetricQueryReplayRejectsTooLargeWindow(t *testing.T) {
	_, err := resolveMetricQuery(
		"rpc_request_rate",
		MetricProfileReplay,
		"2026-03-20T00:00:00Z",
		"2026-03-22T00:00:01Z",
	)
	if err == nil {
		t.Fatal("expected replay query to reject ranges larger than 24h")
	}
}

func TestParseMetricProfile(t *testing.T) {
	profile, err := ParseMetricProfile("", false)
	if err != nil {
		t.Fatalf("ParseMetricProfile failed: %v", err)
	}
	if profile != MetricProfileOverview {
		t.Fatalf("default profile = %s, want %s", profile, MetricProfileOverview)
	}

	if _, err := ParseMetricProfile(string(MetricProfileReplay), false); err == nil {
		t.Fatal("expected replay to be rejected for snapshot endpoint")
	}
}

func contains(s, want string) bool {
	return len(s) >= len(want) && (s == want || len(want) > 0 && (indexOf(s, want) >= 0))
}

func indexOf(s, want string) int {
	for i := 0; i+len(want) <= len(s); i++ {
		if s[i:i+len(want)] == want {
			return i
		}
	}
	return -1
}

package catalog

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"flashsale/ops/backend/model"
)

type MetricProfile string

const (
	MetricProfileOverview MetricProfile = "overview"
	MetricProfileBurst    MetricProfile = "burst"
	MetricProfileReplay   MetricProfile = "replay"
)

const replayRangeLimit = 24 * time.Hour

type metricCatalogEntry struct {
	Def             model.MetricDef
	OverviewQuery   string
	BurstQuery      string
	ReplayQueryFunc func(start, end string) (string, error)
}

var metricCatalog = []metricCatalogEntry{
	{
		Def: model.MetricDef{
			Name:   "rpc_request_rate",
			Unit:   "req/s",
			Desc:   "RPC request rate",
			Format: "scalar",
		},
		OverviewQuery: `sum(rate(rpc_server_requests_duration_ms_count[5m]))`,
		BurstQuery:    `sum(rate(rpc_server_requests_duration_ms_count[30s]))`,
		ReplayQueryFunc: func(start, end string) (string, error) {
			lookback, err := replayLookback(start, end)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf(`sum(rate(rpc_server_requests_duration_ms_count[%s]))`, lookback), nil
		},
	},
	{
		Def: model.MetricDef{
			Name:   "rpc_request_rate_by_service",
			Unit:   "req/s",
			Desc:   "RPC request rate by service",
			Format: "vector",
		},
		OverviewQuery: `sum by (job) (rate(rpc_server_requests_duration_ms_count[5m]))`,
		BurstQuery:    `sum by (job) (rate(rpc_server_requests_duration_ms_count[30s]))`,
		ReplayQueryFunc: func(start, end string) (string, error) {
			lookback, err := replayLookback(start, end)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf(`sum by (job) (rate(rpc_server_requests_duration_ms_count[%s]))`, lookback), nil
		},
	},
	{
		Def: model.MetricDef{
			Name:   "rpc_p99_latency",
			Unit:   "ms",
			Desc:   "RPC p99 latency",
			Format: "scalar",
		},
		OverviewQuery: `histogram_quantile(0.99, sum by (le) (rate(rpc_server_requests_duration_ms_bucket[5m])))`,
		BurstQuery:    `histogram_quantile(0.99, sum by (le) (rate(rpc_server_requests_duration_ms_bucket[30s])))`,
		ReplayQueryFunc: func(start, end string) (string, error) {
			lookback, err := replayLookback(start, end)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf(`histogram_quantile(0.99, sum by (le) (rate(rpc_server_requests_duration_ms_bucket[%s])))`, lookback), nil
		},
	},
	{
		Def: model.MetricDef{
			Name:   "rpc_p99_latency_by_service",
			Unit:   "ms",
			Desc:   "RPC p99 latency by service",
			Format: "vector",
		},
		OverviewQuery: `histogram_quantile(0.99, sum by (job, le) (rate(rpc_server_requests_duration_ms_bucket[5m])))`,
		BurstQuery:    `histogram_quantile(0.99, sum by (job, le) (rate(rpc_server_requests_duration_ms_bucket[30s])))`,
		ReplayQueryFunc: func(start, end string) (string, error) {
			lookback, err := replayLookback(start, end)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf(`histogram_quantile(0.99, sum by (job, le) (rate(rpc_server_requests_duration_ms_bucket[%s])))`, lookback), nil
		},
	},
	{
		Def: model.MetricDef{
			Name:   "rpc_error_rate",
			Unit:   "%",
			Desc:   "RPC system error rate",
			Format: "scalar",
		},
		OverviewQuery: `(sum(rate(rpc_server_requests_code_total{code=~"Internal|Unavailable|DeadlineExceeded|Unknown|DataLoss"}[5m])) or vector(0)) / clamp_min(sum(rate(rpc_server_requests_code_total[5m])), 1) * 100`,
		BurstQuery:    `(sum(rate(rpc_server_requests_code_total{code=~"Internal|Unavailable|DeadlineExceeded|Unknown|DataLoss"}[30s])) or vector(0)) / clamp_min(sum(rate(rpc_server_requests_code_total[30s])), 1) * 100`,
		ReplayQueryFunc: func(start, end string) (string, error) {
			lookback, err := replayLookback(start, end)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf(`(sum(rate(rpc_server_requests_code_total{code=~"Internal|Unavailable|DeadlineExceeded|Unknown|DataLoss"}[%s])) or vector(0)) / clamp_min(sum(rate(rpc_server_requests_code_total[%s])), 1) * 100`, lookback, lookback), nil
		},
	},
	{
		Def: model.MetricDef{
			Name:   "goroutines",
			Unit:   "",
			Desc:   "Total goroutines",
			Format: "scalar",
		},
		OverviewQuery: `sum(go_goroutines{job=~"flashsale-.+"})`,
		BurstQuery:    `sum(go_goroutines{job=~"flashsale-.+"})`,
		ReplayQueryFunc: func(_ string, _ string) (string, error) {
			return `sum(go_goroutines{job=~"flashsale-.+"})`, nil
		},
	},
	{
		Def: model.MetricDef{
			Name:   "goroutines_by_service",
			Unit:   "",
			Desc:   "Goroutines by service",
			Format: "vector",
		},
		OverviewQuery: `sum by (job) (go_goroutines{job=~"flashsale-.+"})`,
		BurstQuery:    `sum by (job) (go_goroutines{job=~"flashsale-.+"})`,
		ReplayQueryFunc: func(_ string, _ string) (string, error) {
			return `sum by (job) (go_goroutines{job=~"flashsale-.+"})`, nil
		},
	},
	{
		Def: model.MetricDef{
			Name:   "heap_bytes",
			Unit:   "bytes",
			Desc:   "Allocated heap bytes",
			Format: "scalar",
		},
		OverviewQuery: `sum(go_memstats_heap_alloc_bytes{job=~"flashsale-.+"})`,
		BurstQuery:    `sum(go_memstats_heap_alloc_bytes{job=~"flashsale-.+"})`,
		ReplayQueryFunc: func(_ string, _ string) (string, error) {
			return `sum(go_memstats_heap_alloc_bytes{job=~"flashsale-.+"})`, nil
		},
	},
	{
		Def: model.MetricDef{
			Name:   "process_cpu_seconds_rate",
			Unit:   "cores",
			Desc:   "Process CPU usage rate",
			Format: "scalar",
		},
		OverviewQuery: `sum(rate(process_cpu_seconds_total{job=~"flashsale-.+"}[1m]))`,
		BurstQuery:    `sum(rate(process_cpu_seconds_total{job=~"flashsale-.+"}[1m]))`,
		ReplayQueryFunc: func(_ string, _ string) (string, error) {
			return `sum(rate(process_cpu_seconds_total{job=~"flashsale-.+"}[1m]))`, nil
		},
	},
	{
		Def: model.MetricDef{
			Name:   "http_request_rate",
			Unit:   "req/s",
			Desc:   "HTTP gateway request rate",
			Format: "scalar",
		},
		OverviewQuery: `sum(rate(http_server_requests_duration_ms_count[1m]))`,
		BurstQuery:    `sum(rate(http_server_requests_duration_ms_count[1m]))`,
		ReplayQueryFunc: func(_ string, _ string) (string, error) {
			return `sum(rate(http_server_requests_duration_ms_count[1m]))`, nil
		},
	},
	{
		Def: model.MetricDef{
			Name:   "seckill_purchase_kafka_publish_rate",
			Unit:   "req/s",
			Desc:   "Seckill purchase Kafka publish rate",
			Format: "scalar",
		},
		OverviewQuery: `sum(rate(seckill_purchase_kafka_published_total[1m]))`,
		BurstQuery:    `sum(rate(seckill_purchase_kafka_published_total[30s]))`,
		ReplayQueryFunc: func(start, end string) (string, error) {
			lookback, err := replayLookback(start, end)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf(`sum(rate(seckill_purchase_kafka_published_total[%s]))`, lookback), nil
		},
	},
	{
		Def: model.MetricDef{
			Name:   "seckill_purchase_kafka_publish_failed",
			Unit:   "",
			Desc:   "Seckill purchase Kafka publish failures in the last minute",
			Format: "scalar",
		},
		OverviewQuery: `sum(increase(seckill_purchase_kafka_publish_failed_total[1m]))`,
		BurstQuery:    `sum(increase(seckill_purchase_kafka_publish_failed_total[1m]))`,
		ReplayQueryFunc: func(_ string, _ string) (string, error) {
			return `sum(increase(seckill_purchase_kafka_publish_failed_total[1m]))`, nil
		},
	},
	{
		Def: model.MetricDef{
			Name:   "seckill_order_state_consume_rate",
			Unit:   "req/s",
			Desc:   "Seckill order-state Kafka consume rate",
			Format: "scalar",
		},
		OverviewQuery: `sum(rate(seckill_order_state_consumed_total[1m]))`,
		BurstQuery:    `sum(rate(seckill_order_state_consumed_total[30s]))`,
		ReplayQueryFunc: func(start, end string) (string, error) {
			lookback, err := replayLookback(start, end)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf(`sum(rate(seckill_order_state_consumed_total[%s]))`, lookback), nil
		},
	},
}

func ParseMetricProfile(raw string, allowReplay bool) (MetricProfile, error) {
	switch strings.TrimSpace(raw) {
	case "", string(MetricProfileOverview):
		return MetricProfileOverview, nil
	case string(MetricProfileBurst):
		return MetricProfileBurst, nil
	case string(MetricProfileReplay):
		if !allowReplay {
			return "", fmt.Errorf("profile %q is not supported for this endpoint", raw)
		}
		return MetricProfileReplay, nil
	default:
		return "", fmt.Errorf("unknown profile: %s", raw)
	}
}

func FindMetricDef(name string) *model.MetricDef {
	for i := range metricCatalog {
		if metricCatalog[i].Def.Name == name {
			return &metricCatalog[i].Def
		}
	}
	return nil
}

func resolveMetricQuery(name string, profile MetricProfile, start, end string) (string, error) {
	for i := range metricCatalog {
		entry := metricCatalog[i]
		if entry.Def.Name != name {
			continue
		}
		switch profile {
		case MetricProfileOverview:
			return entry.OverviewQuery, nil
		case MetricProfileBurst:
			return entry.BurstQuery, nil
		case MetricProfileReplay:
			return entry.ReplayQueryFunc(start, end)
		default:
			return "", fmt.Errorf("unsupported profile: %s", profile)
		}
	}
	return "", fmt.Errorf("unknown metric: %s", name)
}

func replayLookback(start, end string) (string, error) {
	startAt, err := parsePromTime(start)
	if err != nil {
		return "", fmt.Errorf("parse start time: %w", err)
	}
	endAt, err := parsePromTime(end)
	if err != nil {
		return "", fmt.Errorf("parse end time: %w", err)
	}
	if !endAt.After(startAt) {
		return "", fmt.Errorf("end must be after start")
	}
	rangeDuration := endAt.Sub(startAt)
	switch {
	case rangeDuration <= 15*time.Minute:
		return "30s", nil
	case rangeDuration <= 2*time.Hour:
		return "1m", nil
	case rangeDuration <= replayRangeLimit:
		return "5m", nil
	default:
		return "", fmt.Errorf("replay range exceeds %s", replayRangeLimit)
	}
}

func parsePromTime(raw string) (time.Time, error) {
	if ts, err := strconv.ParseFloat(raw, 64); err == nil {
		sec, frac := mathModf(ts)
		return time.Unix(sec, frac).UTC(), nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time %q", raw)
	}
	return parsed.UTC(), nil
}

func mathModf(v float64) (int64, int64) {
	sec := int64(v)
	nsec := int64((v - float64(sec)) * float64(time.Second))
	return sec, nsec
}

type cacheEntry struct {
	data      json.RawMessage
	fetchedAt time.Time
}

var (
	cacheMu  sync.RWMutex
	cacheMap = make(map[string]cacheEntry)
	cacheTTL = 2 * time.Second
)

func GetCachedOrFetch(key string, fetch func() (json.RawMessage, error)) (json.RawMessage, error) {
	cacheMu.RLock()
	if entry, ok := cacheMap[key]; ok && time.Since(entry.fetchedAt) < cacheTTL {
		cacheMu.RUnlock()
		return entry.data, nil
	}
	cacheMu.RUnlock()

	data, err := fetch()
	if err != nil {
		return nil, err
	}

	cacheMu.Lock()
	cacheMap[key] = cacheEntry{data: data, fetchedAt: time.Now()}
	cacheMu.Unlock()

	return data, nil
}

var promHTTPClient = &http.Client{Timeout: 5 * time.Second}

func PromInstantQuery(baseURL, query string) (json.RawMessage, error) {
	u := fmt.Sprintf("%s/api/v1/query?query=%s", baseURL, url.QueryEscape(query))
	return doPromRequest(u)
}

func PromRangeQuery(baseURL, query, start, end, step string) (json.RawMessage, error) {
	u := fmt.Sprintf("%s/api/v1/query_range?query=%s&start=%s&end=%s&step=%s",
		baseURL,
		url.QueryEscape(query),
		url.QueryEscape(start),
		url.QueryEscape(end),
		url.QueryEscape(step),
	)
	return doPromRequest(u)
}

func doPromRequest(rawURL string) (json.RawMessage, error) {
	resp, err := promHTTPClient.Get(rawURL)
	if err != nil {
		return nil, fmt.Errorf("prometheus request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading prometheus response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prometheus returned %d: %s", resp.StatusCode, string(body))
	}

	return json.RawMessage(body), nil
}

func MetricsCatalogItems() []map[string]string {
	items := make([]map[string]string, 0, len(metricCatalog))
	for _, metric := range metricCatalog {
		items = append(items, map[string]string{
			"name":   metric.Def.Name,
			"unit":   metric.Def.Unit,
			"desc":   metric.Def.Desc,
			"format": metric.Def.Format,
		})
	}
	return items
}

func MetricsSnapshot(env *EnvContext, names []string, profile MetricProfile) map[string]any {
	results := make(map[string]any, len(names))
	promBase := env.PrometheusURL

	for _, name := range names {
		name = strings.TrimSpace(name)
		if FindMetricDef(name) == nil {
			results[name] = map[string]any{"error": "unknown metric"}
			continue
		}

		query, err := resolveMetricQuery(name, profile, "", "")
		if err != nil {
			results[name] = map[string]any{"error": err.Error()}
			continue
		}

		cacheKey := fmt.Sprintf("instant:%s:%s", profile, name)
		data, err := GetCachedOrFetch(cacheKey, func() (json.RawMessage, error) {
			return PromInstantQuery(promBase, query)
		})
		if err != nil {
			results[name] = map[string]any{"error": err.Error()}
			continue
		}
		results[name] = json.RawMessage(data)
	}

	return results
}

func MetricsRange(env *EnvContext, name, start, end, step string, profile MetricProfile) (json.RawMessage, error) {
	if FindMetricDef(name) == nil {
		return nil, fmt.Errorf("unknown metric: %s", name)
	}

	query, err := resolveMetricQuery(name, profile, start, end)
	if err != nil {
		return nil, err
	}

	cacheKey := fmt.Sprintf("range:%s:%s:%s:%s:%s", profile, name, start, end, step)
	return GetCachedOrFetch(cacheKey, func() (json.RawMessage, error) {
		return PromRangeQuery(env.PrometheusURL, query, start, end, step)
	})
}

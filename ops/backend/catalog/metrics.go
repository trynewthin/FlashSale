package catalog

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"flashsale/ops/backend/model"
)

// ─── Metric Allowlist ───

// MetricAllowlist 预定义的指标列表。
var MetricAllowlist = []model.MetricDef{
	{
		Name:   "rpc_request_rate",
		Query:  `sum(rate(rpc_server_requests_duration_ms_count[5m]))`,
		Unit:   "req/s",
		Desc:   "RPC 整体请求速率",
		Format: "scalar",
	},
	{
		Name:   "rpc_request_rate_by_service",
		Query:  `sum by (job) (rate(rpc_server_requests_duration_ms_count[5m]))`,
		Unit:   "req/s",
		Desc:   "按服务的 RPC 请求速率",
		Format: "vector",
	},
	{
		Name:   "rpc_p99_latency",
		Query:  `histogram_quantile(0.99, sum by (le) (rate(rpc_server_requests_duration_ms_bucket[5m])))`,
		Unit:   "ms",
		Desc:   "RPC P99 延迟",
		Format: "scalar",
	},
	{
		Name:   "rpc_p99_latency_by_service",
		Query:  `histogram_quantile(0.99, sum by (job, le) (rate(rpc_server_requests_duration_ms_bucket[5m])))`,
		Unit:   "ms",
		Desc:   "按服务的 RPC P99 延迟",
		Format: "vector",
	},
	{
		Name:   "rpc_error_rate",
		Query:  `(sum(rate(rpc_server_requests_code_total{code=~"Internal|Unavailable|DeadlineExceeded|Unknown|DataLoss"}[5m])) or vector(0)) / clamp_min(sum(rate(rpc_server_requests_code_total[5m])), 1) * 100`,
		Unit:   "%",
		Desc:   "RPC 系统异常比例（排除业务拒绝）",
		Format: "scalar",
	},
	{
		Name:   "goroutines",
		Query:  `sum(go_goroutines{job=~"flashsale-.+"})`,
		Unit:   "",
		Desc:   "Go 协程总数",
		Format: "scalar",
	},
	{
		Name:   "goroutines_by_service",
		Query:  `sum by (job) (go_goroutines{job=~"flashsale-.+"})`,
		Unit:   "",
		Desc:   "按服务的 Go 协程数",
		Format: "vector",
	},
	{
		Name:   "heap_bytes",
		Query:  `sum(go_memstats_heap_alloc_bytes{job=~"flashsale-.+"})`,
		Unit:   "bytes",
		Desc:   "Go 堆内存总量",
		Format: "scalar",
	},
	{
		Name:   "process_cpu_seconds_rate",
		Query:  `sum(rate(process_cpu_seconds_total{job=~"flashsale-.+"}[1m]))`,
		Unit:   "cores",
		Desc:   "进程级 CPU 使用率",
		Format: "scalar",
	},
	{
		Name:   "http_request_rate",
		Query:  `sum(rate(http_server_requests_duration_ms_count[1m]))`,
		Unit:   "req/s",
		Desc:   "HTTP 网关请求速率",
		Format: "scalar",
	},
	{
		Name:   "seckill_purchase_task_queue_depth",
		Query:  `seckill_purchase_task_queue_depth`,
		Unit:   "",
		Desc:   "秒杀异步购买任务队列当前深度",
		Format: "scalar",
	},
	{
		Name:   "seckill_purchase_task_queue_cap",
		Query:  `seckill_purchase_task_queue_cap`,
		Unit:   "",
		Desc:   "秒杀异步购买任务队列容量",
		Format: "scalar",
	},
	{
		Name:   "seckill_purchase_task_dropped_total",
		Query:  `sum(increase(seckill_purchase_task_dropped_total[1m]))`,
		Unit:   "",
		Desc:   "秒杀异步购买任务丢弃数（过去 1 分钟）",
		Format: "scalar",
	},
}

// FindMetricDef 按名称查找指标定义。
func FindMetricDef(name string) *model.MetricDef {
	for i := range MetricAllowlist {
		if MetricAllowlist[i].Name == name {
			return &MetricAllowlist[i]
		}
	}
	return nil
}

// ─── TTL 缓存 ───

type cacheEntry struct {
	data      json.RawMessage
	fetchedAt time.Time
}

var (
	cacheMu  sync.RWMutex
	cacheMap = make(map[string]cacheEntry)
	cacheTTL = 2 * time.Second
)

// GetCachedOrFetch 带 TTL 缓存的数据获取。
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

// ─── Prometheus 查询 ───

var promHTTPClient = &http.Client{Timeout: 5 * time.Second}

// PromInstantQuery 执行 Prometheus instant query。
func PromInstantQuery(baseURL, query string) (json.RawMessage, error) {
	u := fmt.Sprintf("%s/api/v1/query?query=%s", baseURL, url.QueryEscape(query))
	return doPromRequest(u)
}

// PromRangeQuery 执行 Prometheus range query。
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

// MetricsCatalogItems 返回指标目录列表（安全，不含 Query）。
func MetricsCatalogItems() []map[string]string {
	items := make([]map[string]string, 0, len(MetricAllowlist))
	for _, m := range MetricAllowlist {
		items = append(items, map[string]string{
			"name":   m.Name,
			"unit":   m.Unit,
			"desc":   m.Desc,
			"format": m.Format,
		})
	}
	return items
}

// MetricsSnapshot 执行 instant query 快照。
func MetricsSnapshot(env *EnvContext, names []string) map[string]any {
	results := make(map[string]any, len(names))
	promBase := env.PrometheusURL

	for _, name := range names {
		name = strings.TrimSpace(name)
		def := FindMetricDef(name)
		if def == nil {
			results[name] = map[string]any{"error": "unknown metric"}
			continue
		}

		cacheKey := "instant:" + name
		data, err := GetCachedOrFetch(cacheKey, func() (json.RawMessage, error) {
			return PromInstantQuery(promBase, def.Query)
		})
		if err != nil {
			results[name] = map[string]any{"error": err.Error()}
			continue
		}
		results[name] = json.RawMessage(data)
	}

	return results
}

// MetricsRange 执行 range query。
func MetricsRange(env *EnvContext, name, start, end, step string) (json.RawMessage, error) {
	def := FindMetricDef(name)
	if def == nil {
		return nil, fmt.Errorf("unknown metric: %s", name)
	}

	cacheKey := fmt.Sprintf("range:%s:%s:%s:%s", name, start, end, step)
	return GetCachedOrFetch(cacheKey, func() (json.RawMessage, error) {
		return PromRangeQuery(env.PrometheusURL, def.Query, start, end, step)
	})
}

// metrics_proxy 提供安全的 Prometheus 查询代理。
// 仅允许预定义的 PromQL（allowlist），防止前端任意查询。
// 内置 TTL 缓存避免高频轮询击穿 Prometheus。
package ops

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// ---------- allowlist ----------

// MetricDef 描述一条预定义指标。
type MetricDef struct {
	Name   string `json:"name"`   // 逻辑名称，如 "rpc_request_rate"
	Query  string `json:"-"`      // PromQL
	Unit   string `json:"unit"`   // 展示单位，如 "req/s"
	Desc   string `json:"desc"`   // 中文说明
	Format string `json:"format"` // "scalar" | "vector" | "matrix"
}

var metricAllowlist = []MetricDef{
	{
		Name:   "rpc_request_rate",
		Query:  `sum(rate(rpc_server_requests_duration_ms_count[1m]))`,
		Unit:   "req/s",
		Desc:   "RPC 整体请求速率",
		Format: "scalar",
	},
	{
		Name:   "rpc_request_rate_by_service",
		Query:  `sum by (job) (rate(rpc_server_requests_duration_ms_count[1m]))`,
		Unit:   "req/s",
		Desc:   "按服务的 RPC 请求速率",
		Format: "vector",
	},
	{
		Name:   "rpc_p99_latency",
		Query:  `histogram_quantile(0.99, sum by (le) (rate(rpc_server_requests_duration_ms_bucket[1m])))`,
		Unit:   "ms",
		Desc:   "RPC P99 延迟",
		Format: "scalar",
	},
	{
		Name:   "rpc_p99_latency_by_service",
		Query:  `histogram_quantile(0.99, sum by (job, le) (rate(rpc_server_requests_duration_ms_bucket[1m])))`,
		Unit:   "ms",
		Desc:   "按服务的 RPC P99 延迟",
		Format: "vector",
	},
	{
		Name:   "rpc_error_rate",
		Query:  `sum(rate(rpc_server_requests_code_total{code!="OK"}[1m])) / sum(rate(rpc_server_requests_code_total[1m])) * 100`,
		Unit:   "%",
		Desc:   "RPC 错误比例",
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
}

func findMetricDef(name string) *MetricDef {
	for i := range metricAllowlist {
		if metricAllowlist[i].Name == name {
			return &metricAllowlist[i]
		}
	}
	return nil
}

// ---------- Prometheus endpoint ----------

func defaultPrometheusEndpoint() string {
	if ep := strings.TrimSpace(os.Getenv("FLASHSALE_PROMETHEUS_ENDPOINT")); ep != "" {
		return ep
	}
	return "http://localhost:9090"
}

// ---------- TTL cache ----------

type cacheEntry struct {
	data      json.RawMessage
	fetchedAt time.Time
}

var (
	cacheMu  sync.RWMutex
	cacheMap = make(map[string]cacheEntry)
	cacheTTL = 2 * time.Second
)

func getCachedOrFetch(key string, fetch func() (json.RawMessage, error)) (json.RawMessage, error) {
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

// ---------- handlers ----------

// GET /api/v1/metrics/catalog — 返回可查询的指标目录
func (s *Server) getMetricsCatalog(w http.ResponseWriter, _ *http.Request) {
	items := make([]map[string]string, 0, len(metricAllowlist))
	for _, m := range metricAllowlist {
		items = append(items, map[string]string{
			"name":   m.Name,
			"unit":   m.Unit,
			"desc":   m.Desc,
			"format": m.Format,
		})
	}
	writeOK(w, map[string]any{"metrics": items})
}

// GET /api/v1/metrics/snapshot?names=rpc_request_rate,goroutines
// 返回 instant query 结果。
func (s *Server) getMetricsSnapshot(w http.ResponseWriter, r *http.Request) {
	namesStr := r.URL.Query().Get("names")
	if namesStr == "" {
		writeErr(w, http.StatusBadRequest, "missing 'names' query parameter")
		return
	}

	names := strings.Split(namesStr, ",")
	results := make(map[string]any, len(names))
	promBase := defaultPrometheusEndpoint()

	for _, name := range names {
		name = strings.TrimSpace(name)
		def := findMetricDef(name)
		if def == nil {
			results[name] = map[string]any{"error": "unknown metric"}
			continue
		}

		cacheKey := "instant:" + name
		data, err := getCachedOrFetch(cacheKey, func() (json.RawMessage, error) {
			return promInstantQuery(promBase, def.Query)
		})
		if err != nil {
			results[name] = map[string]any{"error": err.Error()}
			continue
		}
		results[name] = json.RawMessage(data)
	}

	writeOK(w, map[string]any{"snapshot": results})
}

// GET /api/v1/metrics/range?name=rpc_request_rate&start=...&end=...&step=15s
// 返回 range query 结果（用于图表）。
func (s *Server) getMetricsRange(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	start := r.URL.Query().Get("start")
	end := r.URL.Query().Get("end")
	step := r.URL.Query().Get("step")

	if name == "" || start == "" || end == "" {
		writeErr(w, http.StatusBadRequest, "missing required query parameters: name, start, end")
		return
	}
	if step == "" {
		step = "15s"
	}

	def := findMetricDef(name)
	if def == nil {
		writeErr(w, http.StatusBadRequest, fmt.Sprintf("unknown metric: %s", name))
		return
	}

	cacheKey := fmt.Sprintf("range:%s:%s:%s:%s", name, start, end, step)
	data, err := getCachedOrFetch(cacheKey, func() (json.RawMessage, error) {
		return promRangeQuery(defaultPrometheusEndpoint(), def.Query, start, end, step)
	})
	if err != nil {
		writeErr(w, http.StatusBadGateway, fmt.Sprintf("prometheus query failed: %v", err))
		return
	}

	writeOK(w, map[string]any{
		"name":   name,
		"unit":   def.Unit,
		"result": json.RawMessage(data),
	})
}

// ---------- Prometheus HTTP queries ----------

func promInstantQuery(baseURL, query string) (json.RawMessage, error) {
	u := fmt.Sprintf("%s/api/v1/query?query=%s", baseURL, url.QueryEscape(query))
	return doPromRequest(u)
}

func promRangeQuery(baseURL, query, start, end, step string) (json.RawMessage, error) {
	u := fmt.Sprintf("%s/api/v1/query_range?query=%s&start=%s&end=%s&step=%s",
		baseURL,
		url.QueryEscape(query),
		url.QueryEscape(start),
		url.QueryEscape(end),
		url.QueryEscape(step),
	)
	return doPromRequest(u)
}

var promHTTPClient = &http.Client{Timeout: 5 * time.Second}

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

	// 返回原始的 Prometheus API JSON response。
	return json.RawMessage(body), nil
}

// Package metrics 提供 Prometheus 指标注册与 HTTP 暴露能力。
package metrics

import (
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsConfig 定义指标初始化参数。
type MetricsConfig struct {
	Namespace string
}

var (
	mu       sync.RWMutex
	registry *prometheus.Registry
)

// Init 初始化指标注册表并注册默认运行时指标。
func Init(cfg MetricsConfig) error {
	mu.Lock()
	defer mu.Unlock()

	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	registry = reg
	return nil
}

// RegisterHTTP 返回用于暴露 /metrics 的 HTTP Handler。
func RegisterHTTP() http.Handler {
	mu.RLock()
	reg := registry
	mu.RUnlock()

	if reg == nil {
		_ = Init(MetricsConfig{})
		mu.RLock()
		reg = registry
		mu.RUnlock()
	}
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
}

package model

// ─── Prometheus 指标 ───

// MetricDef 描述一条预定义指标。
type MetricDef struct {
	Name   string `json:"name"`   // 逻辑名称，如 "rpc_request_rate"
	Query  string `json:"-"`      // PromQL
	Unit   string `json:"unit"`   // 展示单位，如 "req/s"
	Desc   string `json:"desc"`   // 中文说明
	Format string `json:"format"` // "scalar" | "vector" | "matrix"
}

// ─── 可观测性 ───

// ObservabilityLink 描述单个可观测性工具的入口。
type ObservabilityLink struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Available bool   `json:"available"`
}

// ObservabilityLinks 聚合所有可观测性工具入口。
type ObservabilityLinks struct {
	Jaeger     ObservabilityLink `json:"jaeger"`
	Prometheus ObservabilityLink `json:"prometheus"`
	Grafana    ObservabilityLink `json:"grafana"`
}

// ObservabilityToolDef 定义工具名称、对应容器服务名、默认容器端口和协议。
type ObservabilityToolDef struct {
	Name          string
	ServiceName   string
	ContainerPort string // 期望暴露的容器端口（如 "16686"）
	Protocol      string // "http"
	EnvOverride   string // 环境变量覆盖 key（如 "FLASHSALE_JAEGER_URL"）
}

// ─── 服务日志 ───

// ServiceLogFile 表示可查看的服务日志文件。
type ServiceLogFile struct {
	ID       string `json:"id"`
	RelPath  string `json:"rel_path"`
	Name     string `json:"name"`
	SizeByte int64  `json:"size_byte"`
	ModUnix  int64  `json:"mod_unix"`
}

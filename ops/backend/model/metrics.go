package model

type MetricDef struct {
	Name   string `json:"name"`
	Query  string `json:"-"`
	Unit   string `json:"unit"`
	Desc   string `json:"desc"`
	Format string `json:"format"`
}

type ObservabilityLink struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Available bool   `json:"available"`
}

type ObservabilityLinks struct {
	Jaeger     ObservabilityLink `json:"jaeger"`
	Prometheus ObservabilityLink `json:"prometheus"`
	Grafana    ObservabilityLink `json:"grafana"`
}

type ObservabilityToolDef struct {
	Name          string
	ServiceName   string
	ContainerPort string
	Protocol      string
	EnvOverride   string
}

type ServiceLogFile struct {
	ID       string `json:"id"`
	RelPath  string `json:"rel_path"`
	Name     string `json:"name"`
	SizeByte int64  `json:"size_byte"`
	ModUnix  int64  `json:"mod_unix"`
}

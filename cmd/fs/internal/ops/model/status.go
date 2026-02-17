package model

// ─── 状态快照 ───

// StatusSnapshot 是 ops-control 的状态快照（用于 Web 面板展示）。
type StatusSnapshot struct {
	NowUnix int64 `json:"now_unix"`

	RepoRoot string `json:"repo_root"`
	// DeploymentMode 表示当前检查口径：
	// host_process=本机直连进程模式；docker_app=容器化部署模式。
	DeploymentMode string `json:"deployment_mode"`

	Git struct {
		Branch string `json:"branch"`
		Commit string `json:"commit"`
		Dirty  bool   `json:"dirty"`
	} `json:"git"`

	Ports []PortStatus `json:"ports"`
	HTTP  []HTTPStatus `json:"http"`

	Docker struct {
		OK         bool              `json:"ok"`
		Error      string            `json:"error,omitempty"`
		Containers []DockerContainer `json:"containers,omitempty"`
		Raw        string            `json:"raw,omitempty"`
	} `json:"docker"`

	Files struct {
		SeedResultPath string `json:"seed_result_path"`
		SeedResultOK   bool   `json:"seed_result_ok"`
	} `json:"files"`
}

// PortStatus 表示端口监听状态。
type PortStatus struct {
	Name string `json:"name"`
	Addr string `json:"addr"`
	OK   bool   `json:"ok"`
}

// HTTPStatus 表示 HTTP 健康检查状态。
type HTTPStatus struct {
	Name       string `json:"name"`
	URL        string `json:"url"`
	OK         bool   `json:"ok"`
	StatusCode int    `json:"status_code"`
}

// DockerContainer 是 docker ps 的简化结构。
type DockerContainer struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Ports  string `json:"ports"`
}

// ─── 部署模式常量 ───

const (
	DeploymentModeHostProcess = "host_process"
	DeploymentModeDockerApp   = "docker_app"
)
